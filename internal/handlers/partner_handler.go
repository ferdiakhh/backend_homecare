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

// AcceptOrder untuk Mitra mengambil job
func AcceptOrder(c *gin.Context) {
	mitraID, _ := c.Get("userID") // ID User login (Mitra)
	orderID := c.Param("id")

	// 1. Cari Ordernya dulu
	var order models.Order
	if err := config.DB.First(&order, orderID).Error; err != nil {
		utils.APIResponse(c, http.StatusNotFound, false, "Order tidak ditemukan", nil)
		return
	}

	// 2. Validasi: Apakah order masih available?
	// Ini mencegah "Race Condition" (Dua perawat klik barengan)
	if order.PartnerID != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Yah, Order ini sudah diambil perawat lain!", nil)
		return
	}

	if order.Status != "PAID" {
		utils.APIResponse(c, http.StatusBadRequest, false, "Order belum lunas / sudah selesai", nil)
		return
	}

	// 3. Cari ID Profile Mitra berdasarkan User ID
	var profile models.PartnerProfile
	if err := config.DB.Where("user_id = ?", mitraID).First(&profile).Error; err != nil {
		utils.APIResponse(c, http.StatusForbidden, false, "Anda belum melengkapi profil mitra", nil)
		return
	}

	// 4. Update Order: Masukkan ID Mitra & Ubah Status
	// Kita pakai Transaction biar aman
	tx := config.DB.Begin()

	// Update Partner ID dan Status jadi ASSIGNED
	order.PartnerID = &profile.ID // Simpan ID Profil Mitra, bukan User ID
	order.Status = "ASSIGNED"     // Status baru: Sudah ada perawat

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal mengambil order", nil)
		return
	}

	tx.Commit()

	utils.APIResponse(c, http.StatusOK, true, "Selamat! Order berhasil diambil. Segera hubungi pasien.", order)
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
