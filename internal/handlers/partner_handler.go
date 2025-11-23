package handlers

import (
	"errors"
	"homecare-backend/internal/config"
	"homecare-backend/internal/models"
	"homecare-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func UpdatePartnerProfile(c *gin.Context) {
	// 1. Ambil User ID dari Middleware
	userID, _ := c.Get("userID")

	// 2. Validasi Input JSON
	var input models.UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Input tidak valid", err.Error())
		return
	}

	// 3. Cari Profil Mitra di DB
	var profile models.PartnerProfile
	err := config.DB.Where("user_id = ?", userID).First(&profile).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// KASUS 1: Profil belum ada -> Buat Baru
			profile = models.PartnerProfile{
				UserID:          userID.(uint64),
				STRNumber:       input.STRNumber,
				ExperienceYears: input.ExperienceYears,
				VideoIntroURL:   input.VideoIntroURL,
				BioDescription:  input.BioDescription,
				CurrentLat:      input.CurrentLat,
				CurrentLng:      input.CurrentLng,
				IsActive:        true, // Langsung aktifkan (atau bisa nunggu verifikasi admin)
			}
			if err := config.DB.Create(&profile).Error; err != nil {
				utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal membuat profil mitra", err.Error())
				return
			}
		} else {
			// Error DB selain record not found
			utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal mengambil profil mitra", err.Error())
			return
		}
	} else {
		// KASUS 2: Profil sudah ada -> Update Data
		if err := config.DB.Model(&profile).Updates(models.PartnerProfile{
			STRNumber:       input.STRNumber,
			ExperienceYears: input.ExperienceYears,
			VideoIntroURL:   input.VideoIntroURL,
			BioDescription:  input.BioDescription,
			CurrentLat:      input.CurrentLat,
			CurrentLng:      input.CurrentLng,
		}).Error; err != nil {
			utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal mengupdate profil mitra", err.Error())
			return
		}

		// reload profile to return fresh data
		if err := config.DB.Where("id = ?", profile.ID).First(&profile).Error; err != nil {
			utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal mengambil profil setelah update", err.Error())
			return
		}
	}

	utils.APIResponse(c, http.StatusOK, true, "Profil Mitra Berhasil Diupdate!", profile)
}

// Tambahan: Handler untuk melihat list layanan (Biar customer bisa liat menu)
func GetServices(c *gin.Context) {
	var services []struct {
		ID          uint    `json:"id"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Price       float64 `json:"price"`
	}
	// Ambil semua layanan dari DB
	config.DB.Table("services").Find(&services)

	utils.APIResponse(c, http.StatusOK, true, "Daftar Layanan", services)
}

// GetAvailableOrders menampilkan job yang sudah dibayar tapi belum ada perawatnya
func GetAvailableOrders(c *gin.Context) {
	var orders []models.Order

	// Logic: Status PAID + PartnerID masih Kosong (NULL)
	// Preload Service & Patient biar perawat tau ini sakit apa & bayarannya berapa
	config.DB.Preload("Service").Preload("Patient").Where("status = ? AND partner_id IS NULL", "PAID").Find(&orders)
	utils.APIResponse(c, http.StatusOK, true, "Daftar Job Tersedia", orders)
}

// Update di: internal/handlers/partner_handler.go

func AcceptOrder(c *gin.Context) {
	mitraID, _ := c.Get("userID") // ID User Login (User ID)
	orderID := c.Param("id")

	// 1. Cari Order
	var order models.Order
	if err := config.DB.First(&order, orderID).Error; err != nil {
		utils.APIResponse(c, http.StatusNotFound, false, "Order tidak ditemukan", nil)
		return
	}

	// 2. Cari Profile Mitra dari User ID yang login
	var profile models.PartnerProfile
	if err := config.DB.Where("user_id = ?", mitraID).First(&profile).Error; err != nil {
		utils.APIResponse(c, http.StatusForbidden, false, "Profil Mitra tidak ditemukan", nil)
		return
	}

	// 3. LOGIKA BARU (Handling Direct Booking vs Open Booking)

	// Jika PartnerID di order sudah ada isinya (Direct Booking)
	if order.PartnerID != nil {
		// Cek: Apakah ID yang tertulis di order ITU SAYA?
		if *order.PartnerID != profile.ID {
			// Kalau bukan saya, berarti ini orderan direct buat orang lain!
			utils.APIResponse(c, http.StatusForbidden, false, "Maaf, Order ini khusus untuk Mitra lain.", nil)
			return
		}
		// Kalau iya (ID sama), berarti saya sedang mengkonfirmasi orderan direct ini. Lanjut!
	} else {
		// Jika PartnerID kosong (Open Booking), berarti siapa cepat dia dapat.
		// Saya akan isi PartnerID dengan ID saya.
		order.PartnerID = &profile.ID
	}

	// 4. Validasi Status (Hanya boleh ambil yang statusnya PAID)
	if order.Status != "PAID" {
		utils.APIResponse(c, http.StatusBadRequest, false, "Order belum dibayar atau sudah diambil", nil)
		return
	}

	// 5. Update Status
	order.Status = "ASSIGNED" // Atau "ON_DUTY" sesuai kesepakatan DB kemarin

	if err := config.DB.Save(&order).Error; err != nil {
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal konfirmasi order", nil)
		return
	}

	utils.APIResponse(c, http.StatusOK, true, "Order Berhasil Dikonfirmasi! Segera berangkat.", order)
}

// Update di: internal/handlers/partner_handler.go

func SearchPartners(c *gin.Context) {
	// 1. Ambil koordinat Customer/Pasien dari Query Param
	// Contoh URL: GET /api/v1/partners/search?lat=-6.200&lng=106.812
	latStr := c.Query("lat")
	lngStr := c.Query("lng")

	if latStr == "" || lngStr == "" {
		utils.APIResponse(c, http.StatusBadRequest, false, "Koordinat (lat/lng) wajib diisi", nil)
		return
	}

	// Convert string ke float64
	latParam := utils.StringToFloat(latStr) // Pastikan kamu punya helper ini atau pakai strconv
	lngParam := utils.StringToFloat(lngStr)

	var partners []models.PartnerProfile

	// 2. Logika Filtering Radius (Haversine Formula MySQL)
	// Angka 6371 adalah jari-jari bumi dalam KM.
	// Query ini menghitung jarak antara (current_lat, current_lng) mitra dengan (latParam, lngParam) user.

	radiusKM := 15 // Kita cari perawat dalam radius 15 KM

	// Query Raw SQL untuk filter jarak & urutkan dari yang terdekat
	err := config.DB.
		Table("partner_profiles").
		Select("partner_profiles.*, (6371 * acos(cos(radians(?)) * cos(radians(current_lat)) * cos(radians(current_lng) - radians(?)) + sin(radians(?)) * sin(radians(current_lat)))) AS distance", latParam, lngParam, latParam).
		Joins("JOIN users ON users.id = partner_profiles.user_id"). // Join ke user biar bisa preload
		Preload("User").                                            // Load data nama/foto
		Where("is_active = ?", true).
		Having("distance < ?", radiusKM). // Hanya yang < 15 KM
		Order("distance ASC").            // Urutkan yang paling dekat duluan
		Find(&partners).Error

	if err != nil {
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal mencari mitra terdekat", err.Error())
		return
	}

	utils.APIResponse(c, http.StatusOK, true, "Rekomendasi Mitra Terdekat", partners)
}

// RejectOrder: Mitra menolak orderan yang ditujukan padanya (Direct Booking)
func RejectOrder(c *gin.Context) {
	mitraID, _ := c.Get("userID")
	orderID := c.Param("id")

	// 1. Cari Order
	var order models.Order
	if err := config.DB.First(&order, orderID).Error; err != nil {
		utils.APIResponse(c, http.StatusNotFound, false, "Order tidak ditemukan", nil)
		return
	}

	// 2. Validasi: Apakah benar order ini ditujukan ke saya?
	var profile models.PartnerProfile
	config.DB.Where("user_id = ?", mitraID).First(&profile)

	if order.PartnerID == nil || *order.PartnerID != profile.ID {
		utils.APIResponse(c, http.StatusForbidden, false, "Anda tidak berhak menolak order ini", nil)
		return
	}

	// 3. Validasi Status
	if order.Status != "PAID" {
		utils.APIResponse(c, http.StatusBadRequest, false, "Hanya order status PAID yang bisa ditolak", nil)
		return
	}

	// 4. Update Status jadi CANCELLED (atau REFUND_NEEDED)
	// Kita set PartnerID jadi NULL lagi biar history-nya jelas atau biarkan terisi untuk audit admin.
	order.Status = "CANCELLED"

	if err := config.DB.Save(&order).Error; err != nil {
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal menolak order", nil)
		return
	}

	// TODO (Nanti): Trigger notifikasi ke Admin/Customer untuk proses Refund

	utils.APIResponse(c, http.StatusOK, true, "Order ditolak. Admin akan memproses refund ke customer.", nil)
}
