package handlers

import (
	"encoding/json"
	"homecare-backend/internal/config"
	"homecare-backend/internal/models"
	"homecare-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Struct input jurnal dari Frontend
type CreateJournalInput struct {
	Vitals struct {
		Tensi string `json:"tensi"`
		Suhu  string `json:"suhu"`
		Nadi  string `json:"nadi"`
	} `json:"vitals"`
	Notes    string `json:"notes"`
	PhotoURL string `json:"photo_url"`
}

func SubmitMedicalJournal(c *gin.Context) {
	orderID := c.Param("id")

	// 1. Validasi Input JSON
	var input CreateJournalInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Input tidak valid", nil)
		return
	}

	// 2. Ambil Data Order Dulu (Penting untuk perhitungan gaji)
	var order models.Order
	if err := config.DB.First(&order, orderID).Error; err != nil {
		utils.APIResponse(c, http.StatusNotFound, false, "Order tidak ditemukan", nil)
		return
	}

	// Cek: Jangan sampai submit jurnal di order yang belum ASSIGNED atau sudah selesai
	if order.Status == "COMPLETED" {
		utils.APIResponse(c, http.StatusBadRequest, false, "Order ini sudah selesai sebelumnya", nil)
		return
	}

	// MULAI TRANSAKSI DATABASE (PENTING!)
	// Kita pakai transaksi biar aman: Kalau update saldo gagal, simpan jurnal juga dibatalkan.
	tx := config.DB.Begin()

	// 3. Simpan Jurnal Medis
	vitalsJSON, _ := json.Marshal(input.Vitals) // Convert struct ke JSON bytes

	journal := models.CareJournal{
		OrderID:    utils.StringToUint64(orderID),
		VitalsData: vitalsJSON, // Masuk ke kolom JSON di DB
		Notes:      input.Notes,
		PhotoURL:   input.PhotoURL,
	}

	if err := tx.Create(&journal).Error; err != nil {
		tx.Rollback() // Batalkan semua
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal simpan jurnal", nil)
		return
	}

	// 4. Update Status Order jadi COMPLETED
	order.Status = "COMPLETED"
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal update status order", nil)
		return
	}

	// ==========================================
	// 5. LOGIKA GAJIAN (AUTO DISBURSEMENT) 💸
	// ==========================================

	// A. Ambil Data Service (Untuk tahu Admin Fee)
	var service models.Service
	if err := tx.First(&service, order.ServiceID).Error; err != nil {
		tx.Rollback()
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal mengambil data layanan", nil)
		return
	}

	// B. Hitung Jatah Mitra
	// Rumus: (Total Bayar User - Admin Fee Aplikasi) * 85%
	basePrice := order.TotalAmount - service.AdminFee
	mitraShare := basePrice * 0.85

	// C. Cari User ID milik Mitra (Karena Wallet nempel di User, bukan di PartnerProfile)
	// Kita butuh tau "Siapa User ID dari Partner yang mengerjakan order ini?"
	var profile models.PartnerProfile
	if order.PartnerID != nil {
		tx.First(&profile, *order.PartnerID)
	} else {
		tx.Rollback()
		utils.APIResponse(c, http.StatusInternalServerError, false, "Data Mitra di order hilang", nil)
		return
	}

	// D. Cari Wallet Mitra (Kalau gak ada, buat baru)
	var wallet models.Wallet
	if err := tx.Where("user_id = ?", profile.UserID).First(&wallet).Error; err != nil {
		// Buat wallet baru jika belum ada
		wallet = models.Wallet{UserID: profile.UserID, Balance: 0}
		tx.Create(&wallet)
	}

	// E. Tambah Saldo (Income)
	wallet.Balance += mitraShare
	if err := tx.Save(&wallet).Error; err != nil {
		tx.Rollback()
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal update saldo mitra", nil)
		return
	}

	// F. Catat Riwayat Transaksi (Mutasi Masuk)
	trx := models.WalletTransaction{
		WalletID: wallet.ID,
		OrderID:  &order.ID,
		Amount:   mitraShare,
		Type:     "INCOME",
		Status:   "SUCCESS",
	}
	if err := tx.Create(&trx).Error; err != nil {
		tx.Rollback()
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal catat mutasi", nil)
		return
	}

	// SELESAI SEMUA: COMMIT TRANSAKSI
	tx.Commit()

	utils.APIResponse(c, http.StatusOK, true, "Laporan Medis Tersimpan & Saldo Masuk ke Dompet", gin.H{
		"journal_id": journal.ID,
		"income":     mitraShare, // Kasih tau mitra dia dapet berapa
		"status":     "COMPLETED",
	})
}
