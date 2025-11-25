package handlers

import (
	"errors"
	"homecare-backend/internal/config"
	"homecare-backend/internal/models"
	"homecare-backend/pkg/utils"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

	// 3. Log webhook received
	log.Printf("[Webhook] Midtrans notification received - OrderID: %s, TransactionStatus: %s, FraudStatus: %s, MappedStatus: %s",
		notification.OrderID, notification.TransactionStatus, notification.FraudStatus, orderStatus)

	// 4. Update Database
	// Cari order berdasarkan Order ID (Midtrans kirim INV-xxxx)
	var order models.Order
	if err := config.DB.Where("order_no = ?", notification.OrderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("[Webhook] Order not found: %s", notification.OrderID)
			utils.APIResponse(c, http.StatusNotFound, false, "Order Not Found", nil)
			return
		}
		log.Printf("[Webhook] DB error fetching order: %v", err)
		utils.APIResponse(c, http.StatusInternalServerError, false, "Database error", err.Error())
		return
	}

	// 5. Jika status berubah, update ke database
	if order.Status != orderStatus {
		log.Printf("[Webhook] Updating order %s status from %s to %s", notification.OrderID, order.Status, orderStatus)
		order.Status = orderStatus
		if err := config.DB.Save(&order).Error; err != nil {
			log.Printf("[Webhook] DB error updating order: %v", err)
			utils.APIResponse(c, http.StatusInternalServerError, false, "Failed to update order", err.Error())
			return
		}
		log.Printf("[Webhook] Order %s status successfully updated to %s", notification.OrderID, orderStatus)
	} else {
		log.Printf("[Webhook] Order %s status unchanged (already %s)", notification.OrderID, orderStatus)
	}

	// 6. Response OK ke Midtrans (Wajib biar Midtrans tau kita udah terima)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
