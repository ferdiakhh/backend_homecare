package handlers

import (
	"homecare-backend/internal/config"
	"homecare-backend/internal/models"
	"homecare-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetMyWallet menampilkan saldo saat ini & riwayat transaksi
func GetMyWallet(c *gin.Context) {
	userID, _ := c.Get("userID")

	// 1. Ambil Data Wallet
	var wallet models.Wallet
	// Preload Transaction history biar sekalian tampil
	if err := config.DB.Preload("Transactions").Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		// Jika belum punya wallet (baru daftar), buatkan wallet kosong
		wallet = models.Wallet{UserID: userID.(uint64), Balance: 0}
		config.DB.Create(&wallet)
	}

	utils.APIResponse(c, http.StatusOK, true, "Dompet Saya", wallet)
}

// RequestWithdrawal mengajukan penarikan dana
func RequestWithdrawal(c *gin.Context) {
	userID, _ := c.Get("userID")
	var input struct {
		Amount float64 `json:"amount" binding:"required,min=10000"` // Minimal tarik 10rb
		Bank   string  `json:"bank" binding:"required"`
		NoRek  string  `json:"no_rek" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Input salah", err.Error())
		return
	}

	// 1. Cek Saldo Cukup Gak?
	var wallet models.Wallet
	if err := config.DB.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		utils.APIResponse(c, http.StatusNotFound, false, "Wallet tidak ditemukan", nil)
		return
	}

	if wallet.Balance < input.Amount {
		utils.APIResponse(c, http.StatusBadRequest, false, "Saldo tidak cukup!", nil)
		return
	}

	// 2. Buat Transaksi (Status: PENDING)
	// Kita kurangi saldo DULU atau NANTI pas diapprove admin?
	// Best Practice: Kurangi DULU (Lock Balance) biar gak ditarik double.
	// Kalau admin tolak, baru balikin saldonya.

	tx := config.DB.Begin()

	// Kurangi Saldo
	wallet.Balance -= input.Amount
	if err := tx.Save(&wallet).Error; err != nil {
		tx.Rollback()
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal update saldo", nil)
		return
	}

	// Catat di History
	transaction := models.WalletTransaction{
		WalletID: wallet.ID,
		Amount:   input.Amount,
		Type:     "WITHDRAWAL", // Enum: INCOME, WITHDRAWAL
		Status:   "PENDING",    // Enum: SUCCESS, PENDING, FAILED
		// OrderID kosong karena ini transaksi manual
	}

	if err := tx.Create(&transaction).Error; err != nil {
		tx.Rollback()
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal simpan transaksi", nil)
		return
	}

	tx.Commit()

	utils.APIResponse(c, http.StatusCreated, true, "Permintaan penarikan berhasil diajukan. Tunggu konfirmasi Admin.", transaction)
}
