package handlers

import (
	"homecare-backend/internal/config"
	"homecare-backend/internal/models"
	"homecare-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AddPatient menambahkan data keluarga baru
func AddPatient(c *gin.Context) {
	userID, _ := c.Get("userID") // Ambil ID Customer yg login

	var input models.CreatePatientInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Input Data Pasien Salah", err.Error())
		return
	}

	patient := models.Patient{
		CustomerID:     userID.(uint64),
		Name:           input.Name,
		DOB:            input.DOB,
		Gender:         input.Gender,
		Weight:         input.Weight,
		MedicalHistory: input.MedicalHistory,
		AddressDetail:  input.AddressDetail,
		Lat:            input.Lat,
		Lng:            input.Lng,
	}

	if err := config.DB.Create(&patient).Error; err != nil {
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal menyimpan pasien", nil)
		return
	}

	utils.APIResponse(c, http.StatusCreated, true, "Data Pasien Berhasil Ditambahkan", patient)
}

// GetMyPatients melihat daftar keluarga saya
func GetMyPatients(c *gin.Context) {
	userID, _ := c.Get("userID")

	var patients []models.Patient
	config.DB.Where("customer_id = ?", userID).Find(&patients)

	utils.APIResponse(c, http.StatusOK, true, "Daftar Pasien Saya", patients)
}
