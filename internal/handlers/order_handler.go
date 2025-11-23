package handlers

import (
	"fmt"
	"homecare-backend/internal/config"
	"homecare-backend/internal/models"
	"homecare-backend/pkg/utils"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

// CreateOrder membuat pesanan baru
func CreateOrder(c *gin.Context) {
	customerID, _ := c.Get("userID")

	// Kita perlu ambil data user lengkap (Nama & Email) untuk dikirim ke Midtrans
	var customer models.User
	config.DB.First(&customer, customerID)

	var input models.CreateOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Input Order Salah", err.Error())
		return
	}

	// 1. Cek Layanan & Ambil Harga
	var service models.Service
	if err := config.DB.First(&service, input.ServiceID).Error; err != nil {
		utils.APIResponse(c, http.StatusNotFound, false, "Layanan tidak ditemukan", nil)
		return
	}

	totalAmount := service.Price + service.AdminFee
	orderNo := fmt.Sprintf("INV-%d", time.Now().Unix()) // Format: INV-17682391
	endTime := input.ScheduleStart.Add(time.Duration(input.DurationHours) * time.Hour)

	var partnerID *uint64
	if input.PartnerID != 0 {
		partnerID = &input.PartnerID
	}

	// 2. Simpan Order ke DB (Status PENDING)
	order := models.Order{
		OrderNo:       orderNo,
		CustomerID:    customerID.(uint64),
		PatientID:     input.PatientID,
		ServiceID:     input.ServiceID,
		TotalAmount:   totalAmount,
		PartnerID:     partnerID,
		Status:        "PENDING_PAYMENT",
		ScheduleStart: input.ScheduleStart,
		ScheduleEnd:   endTime,
	}

	if err := config.DB.Create(&order).Error; err != nil {
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal menyimpan order", err.Error())
		return
	}

	// ==========================================
	// 3. INTEGRASI MIDTRANS SNAP (BAGIAN BARU)
	// ==========================================

	// A. Init Client Midtrans
	var s = snap.Client{}
	s.New(os.Getenv("MIDTRANS_SERVER_KEY"), midtrans.Sandbox)

	// B. Siapkan Request Snap
	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  orderNo,
			GrossAmt: int64(totalAmount), // Midtrans minta int64
		},
		CreditCard: &snap.CreditCardDetails{
			Secure: true,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: customer.FullName,
			Email: customer.Email,
			Phone: customer.Phone,
		},
		Items: &[]midtrans.ItemDetails{
			{
				ID:    fmt.Sprintf("SVC-%d", service.ID),
				Name:  service.Name,
				Price: int64(service.Price + service.AdminFee),
				Qty:   1,
			},
		},
	}

	// C. Minta Token ke Midtrans
	snapResp, errSnap := s.CreateTransaction(req)
	if errSnap != nil {
		utils.APIResponse(c, http.StatusInternalServerError, false, "Midtrans Error", errSnap.GetMessage())
		return
	}

	// 4. Return Response dengan Token
	utils.APIResponse(c, http.StatusCreated, true, "Order Berhasil! Silakan Bayar.", gin.H{
		"order_id":     order.ID,
		"order_no":     order.OrderNo,
		"total_amount": order.TotalAmount,
		"snap_token":   snapResp.Token,       // <--- INI YG DIPAKAI FRONTEND
		"redirect_url": snapResp.RedirectURL, // <--- Link pembayaran web
	})
}

// GetMyOrders history pesanan customer
func GetMyOrders(c *gin.Context) {
	userID, _ := c.Get("userID")

	var orders []models.Order
	// Preload biar data Service dan Patient ikut keambil
	config.DB.
		Preload("Service").
		Preload("Patient").
		Preload("PartnerProfile.User"). // <--- INI KUNCINYA
		Where("customer_id = ?", userID).
		Order("created_at desc").
		Find(&orders)

	utils.APIResponse(c, http.StatusOK, true, "History Order", orders)
}

// BARU: GetOrderDetail untuk melihat detail + Laporan Medis
func GetOrderDetail(c *gin.Context) {
	userID, _ := c.Get("userID") // ID Customer Login
	orderID := c.Param("id")

	var order models.Order

	// Kita ambil Order spesifik, lalu Preload Jurnal Medis-nya
	err := config.DB.
		Preload("Service").
		Preload("Patient").
		Preload("PartnerProfile.User").
		Preload("CareJournal").                               // <--- Ambil Laporan Medis
		Where("id = ? AND customer_id = ?", orderID, userID). // Pastikan ini order milik dia sendiri
		First(&order).Error

	if err != nil {
		utils.APIResponse(c, http.StatusNotFound, false, "Order tidak ditemukan", nil)
		return
	}

	utils.APIResponse(c, http.StatusOK, true, "Detail Order & Laporan", order)
}
