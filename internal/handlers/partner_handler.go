package handlers

import (
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

	if err == gorm.ErrRecordNotFound {
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
		config.DB.Create(&profile)
	} else {
		// KASUS 2: Profil sudah ada -> Update Data
		config.DB.Model(&profile).Updates(models.PartnerProfile{
			STRNumber:       input.STRNumber,
			ExperienceYears: input.ExperienceYears,
			VideoIntroURL:   input.VideoIntroURL,
			BioDescription:  input.BioDescription,
			CurrentLat:      input.CurrentLat,
			CurrentLng:      input.CurrentLng,
		})
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
