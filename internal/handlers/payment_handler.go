package handlers

import (
	"errors"
	"fmt"
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

	// 7. KIRIM NOTIFIKASI JIKA PAID (NEW ORDER)
	if orderStatus == "PAID" {
		// A. Notifikasi ke Customer (Payment Success)
		var customer models.User
		if err := config.DB.First(&customer, order.CustomerID).Error; err == nil {
			if customer.FCMToken != "" {
				utils.SendNotification(
					customer.FCMToken,
					"Pembayaran Berhasil! ✅",
					"Terima kasih! Pembayaran Anda telah diterima. Kami sedang mencarikan Mitra untuk Anda.",
					map[string]string{"order_id": fmt.Sprintf("%d", order.ID), "type": "payment_success"},
				)
			}
		}

		// B. Cek Direct Booking atau Open Booking
		if order.PartnerID != nil {
			// --- DIRECT BOOKING ---
			// Kita butuh User ID dari partner_profile, tapi di Order cuma ada PartnerID (Profile ID)
			// Jadi kita harus join ke tabel partner_profiles lalu ke users
			var profile models.PartnerProfile
			if err := config.DB.Preload("User").First(&profile, *order.PartnerID).Error; err == nil {
				// Kirim Notif ke Mitra
				if profile.User.FCMToken != "" {
					utils.SendNotification(
						profile.User.FCMToken,
						"Order Baru Masuk! 🔔",
						"Ada pasien yang memesan jasa Anda secara khusus. Segera konfirmasi!",
						map[string]string{"order_id": fmt.Sprintf("%d", order.ID), "type": "new_order_direct"},
					)
				}
			}
		} else {
			// --- OPEN BOOKING (BROADCAST) ---
			// Cari mitra di sekitar pasien (Logic mirip SearchPartners)
			// Kita butuh koordinat pasien. Asumsi: Pasien ada di alamat yg tersimpan (Next: Order harus punya lat/lng sendiri)
			// Untuk sekarang, kita broadcast ke SEMUA mitra yang ONLINE saja dulu atau radius jika memungkinkan.
			// Simplifikasi: Broadcast ke semua mitra aktif yang punya token.

			var activePartners []models.PartnerProfile
			config.DB.Preload("User").Where("is_active = ?", true).Find(&activePartners)

			for _, p := range activePartners {
				if p.User.FCMToken != "" {
					go utils.SendNotification( // Pakai goroutine biar gak blocking
						p.User.FCMToken,
						"Lowongan Job Baru! 📢",
						"Ada order baru di area sekitar Anda. Cek sekarang sebelum diambil orang lain!",
						map[string]string{"order_id": fmt.Sprintf("%d", order.ID), "type": "new_order_open"},
					)
				}
			}
		}
	} else if orderStatus == "CANCELLED" {
		// 8. KIRIM NOTIFIKASI JIKA CANCELLED (Payment Failed/Expired)
		// Cari User Customer
		var customer models.User
		if err := config.DB.First(&customer, order.CustomerID).Error; err == nil {
			if customer.FCMToken != "" {
				utils.SendNotification(
					customer.FCMToken,
					"Pembayaran Gagal/Expired ❌",
					"Maaf, pesanan Anda dibatalkan karena pembayaran gagal atau waktu habis.",
					map[string]string{"order_id": fmt.Sprintf("%d", order.ID), "type": "order_cancelled"},
				)
			}
		}
	}

	// 6. Response OK ke Midtrans (Wajib biar Midtrans tau kita udah terima)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
