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

// GetPatientHistory: Melihat rekam medis/riwayat tindakan satu pasien
func GetPatientHistory(c *gin.Context) {
	userID, _ := c.Get("userID") // ID Customer
	patientID := c.Param("id")

	// 1. Validasi: Pastikan Pasien ini benar milik User yang login
	// (Mencegah user A mengintip data medis pasien user B)
	var patient models.Patient
	if err := config.DB.Where("id = ? AND customer_id = ?", patientID, userID).First(&patient).Error; err != nil {
		utils.APIResponse(c, http.StatusNotFound, false, "Pasien tidak ditemukan atau bukan milik Anda", nil)
		return
	}

	// 2. Ambil History Order + Jurnal Medis
	// Kita cari Order yang PatientID-nya cocok, dan Statusnya sudah COMPLETED
	var histories []models.Order

	err := config.DB.
		Preload("Service").             // Biar tau ini tindakan apa (Infus/Cek Gula/dll)
		Preload("CareJournal").         // <--- INI YG PENTING (Data Medis)
		Preload("PartnerProfile.User"). // Biar tau siapa perawat yg ngerjain
		Where("patient_id = ? AND status = ?", patientID, "COMPLETED").
		Order("schedule_start desc"). // Urutkan dari yang terbaru
		Find(&histories).Error

	if err != nil {
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal memuat riwayat medis", nil)
		return
	}

	// 3. Return Data
	utils.APIResponse(c, http.StatusOK, true, "Rekam Medis Pasien", gin.H{
		"patient_info": patient,
		"history":      histories,
	})
}
