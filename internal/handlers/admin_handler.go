package handlers

import (
	"homecare-backend/internal/config"
	"homecare-backend/internal/models"
	"homecare-backend/pkg/utils"
	"net/http"

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

// UpdateServicePrice mengubah harga layanan
func UpdateServicePrice(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Price    float64 `json:"price"`
		AdminFee float64 `json:"admin_fee"`
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

	// Update
	if input.Price > 0 {
		service.Price = input.Price
	}
	if input.AdminFee > 0 {
		service.AdminFee = input.AdminFee
	}

	config.DB.Save(&service)

	utils.APIResponse(c, http.StatusOK, true, "Harga Layanan Diupdate", service)
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

// GetPendingWithdrawals melihat request tarik dana
func GetPendingWithdrawals(c *gin.Context) {
	var withdrawals []models.WalletTransaction

	config.DB.
		Where("type = ? AND status = ?", "WITHDRAWAL", "PENDING").
		Order("created_at asc").
		Find(&withdrawals)

	utils.APIResponse(c, http.StatusOK, true, "Daftar Penarikan Pending", withdrawals)
}

// ApproveWithdrawal menyetujui transfer
func ApproveWithdrawal(c *gin.Context) {
	trxID := c.Param("id")
	var input struct {
		Action string `json:"action" binding:"required,oneof=approve reject"`
	}
	// Bind JSON...

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

	if input.Action == "approve" {
		trx.Status = "SUCCESS"
		// Di dunia nyata: Panggil API Disbursement Midtrans/Xendit di sini
	} else {
		trx.Status = "FAILED"
		// KEMBALIKAN SALDO KE DOMPET MITRA
		var wallet models.Wallet
		if err := tx.First(&wallet, trx.WalletID).Error; err == nil {
			wallet.Balance += trx.Amount
			tx.Save(&wallet)
		}
	}

	tx.Save(&trx)
	tx.Commit()

	utils.APIResponse(c, http.StatusOK, true, "Status Penarikan Diupdate", nil)
}
