package handlers

import (
	"homecare-backend/internal/config"
	"homecare-backend/internal/models"
	"homecare-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Struct sederhana untuk menangkap body notifikasi Midtrans
// Midtrans mengirim JSON banyak field, tapi kita cuma butuh ini dulu
type MidtransNotification struct {
	TransactionStatus string `json:"transaction_status"`
	OrderID           string `json:"order_id"`
	FraudStatus       string `json:"fraud_status"`
}

func HandleMidtransNotification(c *gin.Context) {
	var notification MidtransNotification

	// 1. Decode JSON dari Midtrans
	if err := c.ShouldBindJSON(&notification); err != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Invalid JSON", nil)
		return
	}

	// 2. Tentukan Status Order Internal berdasarkan Status Midtrans
	var orderStatus string

	switch notification.TransactionStatus {
	case "capture":
		if notification.FraudStatus == "challenge" {
			orderStatus = "PENDING_PAYMENT" // Masih diverifikasi bank
		} else if notification.FraudStatus == "accept" {
			orderStatus = "PAID" // Sukses CC
		}
	case "settlement":
		orderStatus = "PAID" // Sukses Transfer Bank/Gopay
	case "deny", "cancel", "expire":
		orderStatus = "CANCELLED" // Gagal
	case "pending":
		orderStatus = "PENDING_PAYMENT"
	default:
		orderStatus = "PENDING_PAYMENT"
	}

	// 3. Update Database
	// Cari order berdasarkan Order ID (Midtrans kirim INV-xxxx)
	var order models.Order
	if err := config.DB.Where("order_no = ?", notification.OrderID).First(&order).Error; err != nil {
		utils.APIResponse(c, http.StatusNotFound, false, "Order Not Found", nil)
		return
	}

	// Jika status berubah jadi PAID, update
	if order.Status != orderStatus {
		order.Status = orderStatus
		config.DB.Save(&order)
	}

	// 4. Response OK ke Midtrans (Wajib biar Midtrans tau kita udah terima)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
