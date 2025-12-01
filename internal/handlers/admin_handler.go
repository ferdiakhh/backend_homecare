package handlers

import (
	"fmt"
	"homecare-backend/internal/config"
	"homecare-backend/internal/models"
	"homecare-backend/pkg/utils"
	"net/http"

	"strings"

	"github.com/gin-gonic/gin"
)

// GetDashboardStats menampilkan ringkasan performa bisnis
func GetDashboardStats(c *gin.Context) {
	var totalIncome float64
	var activePartners int64
	var ongoingOrders int64
	var pendingWithdrawals int64

	// 1. Hitung Total Pendapatan (Admin Fee)
	// Kita bisa hitung dari tabel orders yang statusnya PAID/COMPLETED
	// Asumsi: Kita ambil sum(admin_fee) dari join services
	// Cara simple: Hitung saldo wallet Admin (Nanti perlu dibuat wallet khusus admin)
	// Atau hitung manual dari transaksi order:
	type Result struct {
		Total float64
	}
	var res Result
	// Query total_amount dari order completed (Ini Gross Revenue)
	config.DB.Table("orders").
		Where("status = ?", "COMPLETED").
		Select("COALESCE(SUM(total_amount), 0) as total"). // Pakai COALESCE biar kalau null jadi 0, dan AS TOTAL biar match struct
		Scan(&res)

	totalIncome = res.Total // Nanti bisa diperbaiki logikanya untuk ambil Net Profit

	// 2. Mitra Aktif
	config.DB.Model(&models.PartnerProfile{}).Where("is_active = ?", true).Count(&activePartners)

	// 3. Order Sedang Berjalan
	config.DB.Model(&models.Order{}).Where("status IN ('PAID', 'ASSIGNED', 'ON_DUTY')").Count(&ongoingOrders)

	// 4. Request Withdraw Pending
	config.DB.Model(&models.WalletTransaction{}).Where("type = ? AND status = ?", "WITHDRAWAL", "PENDING").Count(&pendingWithdrawals)

	utils.APIResponse(c, http.StatusOK, true, "Data Dashboard Admin", gin.H{
		"gross_revenue":         totalIncome,
		"active_partners_count": activePartners,
		"ongoing_orders_count":  ongoingOrders,
		"pending_withdrawals":   pendingWithdrawals,
	})
}

// GetAllCustomers melihat daftar semua user customer
func GetAllCustomers(c *gin.Context) {
	var customers []models.User

	// Role ID 4 = Customer (Sesuaikan dengan DB kamu)
	// Preload Patient biar admin tau customer ini punya pasien siapa aja
	config.DB.
		Preload("Patients").
		Where("role_id = ?", 4).
		Find(&customers)

	utils.APIResponse(c, http.StatusOK, true, "Data Semua Customer", customers)
}

// GetAllOrders melihat semua riwayat transaksi di sistem
func GetAllOrders(c *gin.Context) {
	// Filter status (Opsional) ?status=PAID
	status := c.Query("status")

	var orders []models.Order
	query := config.DB.
		Preload("Service").
		Preload("Patient").
		Preload("PartnerProfile.User"). // Nama Suster
		Preload("Customer").            // Nama Customer (Perlu tambah relasi di model Order)
		Order("created_at desc")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	query.Find(&orders)

	utils.APIResponse(c, http.StatusOK, true, "Data Semua Order", orders)
}

// CreateService menambahkan layanan baru
func CreateService(c *gin.Context) {
	var input struct {
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		Price       float64 `json:"price" binding:"required"`
		AdminFee    float64 `json:"admin_fee"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Input layanan tidak lengkap", err.Error())
		return
	}

	service := models.Service{
		Name:        input.Name,
		Description: input.Description,
		Price:       input.Price,
		AdminFee:    input.AdminFee,
	}

	if err := config.DB.Create(&service).Error; err != nil {
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal membuat layanan", nil)
		return
	}

	utils.APIResponse(c, http.StatusCreated, true, "Layanan Baru Berhasil Dibuat", service)
}

// DeleteService menghapus layanan berdasarkan ID
func DeleteService(c *gin.Context) {
	id := c.Param("id")

	// Cek dulu apakah layanan ini sudah pernah dipesan?
	// Kalau sudah ada history order, sebaiknya jangan dihapus permanen (Hard Delete) karena akan merusak history.
	// Gunakan Soft Delete jika model Service mendukung gorm.DeletedAt, atau tolak penghapusan.

	var count int64
	config.DB.Model(&models.Order{}).Where("service_id = ?", id).Count(&count)
	if count > 0 {
		utils.APIResponse(c, http.StatusBadRequest, false, "Layanan tidak bisa dihapus karena sudah pernah dipesan. Silakan nonaktifkan saja (Fitur next phase) atau update harga.", nil)
		return
	}

	if err := config.DB.Delete(&models.Service{}, id).Error; err != nil {
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal menghapus layanan", nil)
		return
	}

	utils.APIResponse(c, http.StatusOK, true, "Layanan Berhasil Dihapus", nil)
}

// UpdateService mengubah data layanan secara lengkap
func UpdateService(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
		AdminFee    float64 `json:"admin_fee"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Input salah", nil)
		return
	}

	var service models.Service
	if err := config.DB.First(&service, id).Error; err != nil {
		utils.APIResponse(c, http.StatusNotFound, false, "Layanan tidak ditemukan", nil)
		return
	}

	// Update fields jika ada input
	if input.Name != "" {
		service.Name = input.Name
	}
	if input.Description != "" {
		service.Description = input.Description
	}
	if input.Price > 0 {
		service.Price = input.Price
	}
	// AdminFee boleh 0, jadi kita tidak cek > 0 (tapi cek input logic di frontend)
	service.AdminFee = input.AdminFee

	config.DB.Save(&service)

	utils.APIResponse(c, http.StatusOK, true, "Data Layanan Diperbarui", service)
}

// === FITUR ADMIN OPS ===

// GetPendingPartners melihat daftar mitra yang belum diverifikasi
func GetPendingPartners(c *gin.Context) {
	var partners []models.PartnerProfile

	// Cari yang is_active = false (atau nanti bisa tambah kolom is_verified di PartnerProfile)
	// Asumsi: Kalau user.is_verified = false, berarti butuh approval
	config.DB.
		Preload("User").
		Joins("JOIN users ON users.id = partner_profiles.user_id").
		Where("users.is_verified = ?", false).
		Find(&partners)

	utils.APIResponse(c, http.StatusOK, true, "Daftar Mitra Pending", partners)
}

// GetAllPartners melihat daftar SEMUA mitra (Aktif/Non-Aktif)
func GetAllPartners(c *gin.Context) {
	var partners []models.PartnerProfile

	// Filter status (optional) ?active=true
	status := c.Query("active")

	query := config.DB.Preload("User")

	if status == "true" {
		query = query.Where("is_active = ?", true)
	} else if status == "false" {
		query = query.Where("is_active = ?", false)
	}

	query.Find(&partners)

	utils.APIResponse(c, http.StatusOK, true, "Data Semua Mitra", partners)
}

// VerifyPartner menyetujui atau menolak mitra
func VerifyPartner(c *gin.Context) {
	partnerID := c.Param("id")
	var input struct {
		Action string `json:"action" binding:"required,oneof=approve reject"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Input salah", nil)
		return
	}

	var profile models.PartnerProfile
	if err := config.DB.Preload("User").First(&profile, partnerID).Error; err != nil {
		utils.APIResponse(c, http.StatusNotFound, false, "Mitra tidak ditemukan", nil)
		return
	}

	if input.Action == "approve" {
		// Update User jadi Verified & Partner jadi Active
		config.DB.Model(&profile).Update("is_active", true)
		config.DB.Model(&profile.User).Update("is_verified", true)

		utils.APIResponse(c, http.StatusOK, true, "Mitra Berhasil Diverifikasi", nil)
	} else {
		// Kalau Reject, mungkin kirim email notifikasi (Nanti)
		// Untuk sekarang biarkan saja atau hapus
		utils.APIResponse(c, http.StatusOK, true, "Mitra Ditolak", nil)
	}
}

// === FITUR FINANCE ===

// GetAllWithdrawals melihat daftar request tarik dana (Bisa filter status)
func GetAllWithdrawals(c *gin.Context) {
	// Ambil query param ?status=PENDING atau ?status=SUCCESS
	status := c.Query("status")

	var withdrawals []models.WalletTransaction

	// Query Dasar: Ambil semua transaksi tipe WITHDRAWAL
	query := config.DB.
		Where("type = ?", "WITHDRAWAL").
		Order("created_at desc") // Urutkan dari terbaru

	// Filter Dinamis: Kalau ada param status, pasang filter
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Find(&withdrawals).Error; err != nil {
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal memuat data withdrawal", nil)
		return
	}

	utils.APIResponse(c, http.StatusOK, true, "Daftar Penarikan Dana", withdrawals)
}

// ApproveWithdrawal menyetujui transfer
func ApproveWithdrawal(c *gin.Context) {
	trxID := c.Param("id")
	var input struct {
		Action string `json:"action" binding:"required,oneof=approve reject"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Input salah", err.Error())
		return
	}

	// Normalisasi input biar aman (approve/Approve/APPROVE dianggap sama)
	action := strings.ToLower(input.Action)

	var trx models.WalletTransaction
	if err := config.DB.First(&trx, trxID).Error; err != nil {
		utils.APIResponse(c, http.StatusNotFound, false, "Transaksi tidak ditemukan", nil)
		return
	}

	if trx.Status != "PENDING" {
		utils.APIResponse(c, http.StatusBadRequest, false, "Transaksi sudah diproses sebelumnya", nil)
		return
	}

	tx := config.DB.Begin()

	if action == "approve" {
		trx.Status = "SUCCESS"
		// Di sini nanti integrasi Disbursement Xendit/Midtrans
	} else if action == "reject" {
		trx.Status = "FAILED"

		// KEMBALIKAN SALDO KE DOMPET MITRA
		var wallet models.Wallet
		if err := tx.First(&wallet, trx.WalletID).Error; err != nil {
			tx.Rollback()
			utils.APIResponse(c, http.StatusInternalServerError, false, "Wallet mitra tidak ditemukan", nil)
			return
		}

		wallet.Balance += trx.Amount
		if err := tx.Save(&wallet).Error; err != nil {
			tx.Rollback()
			utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal refund saldo", err.Error())
			return
		}
	} else {
		// Jaga-jaga kalau lolos binding tapi bukan approve/reject
		tx.Rollback()
		utils.APIResponse(c, http.StatusBadRequest, false, "Aksi tidak dikenali", nil)
		return
	}

	// PERBAIKAN UTAMA: Cek Error saat Save Status Transaksi!
	if err := tx.Save(&trx).Error; err != nil {
		tx.Rollback()
		// Error ini biasanya karena ENUM di database tidak cocok dengan string "SUCCESS"/"FAILED"
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal update status transaksi: "+err.Error(), nil)
		return
	}

	tx.Commit()

	utils.APIResponse(c, http.StatusOK, true, "Status Penarikan Berhasil Diupdate menjadi "+trx.Status, nil)

	// KIRIM NOTIFIKASI KE MITRA
	// Kita perlu ambil User ID dari Wallet untuk dapat FCM Token
	var wallet models.Wallet
	if err := config.DB.Preload("User").First(&wallet, trx.WalletID).Error; err == nil {
		if wallet.User.FCMToken != "" {
			title := ""
			body := ""
			typeNotif := ""

			if trx.Status == "SUCCESS" {
				title = "Penarikan Dana Berhasil! 💰"
				body = fmt.Sprintf("Permintaan penarikan dana sebesar Rp %.0f telah disetujui dan ditransfer.", trx.Amount)
				typeNotif = "withdrawal_approved"
			} else if trx.Status == "FAILED" {
				title = "Penarikan Dana Ditolak ❌"
				body = "Maaf, permintaan penarikan dana Anda ditolak. Saldo telah dikembalikan."
				typeNotif = "withdrawal_rejected"
			}

			if title != "" {
				utils.SendNotification(
					wallet.User.FCMToken,
					title,
					body,
					map[string]string{"transaction_id": fmt.Sprintf("%d", trx.ID), "type": typeNotif},
				)
			}
		}
	}
}
