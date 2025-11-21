package handlers

import (
	"encoding/json"
	"homecare-backend/internal/config"
	"homecare-backend/internal/models"
	"homecare-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Struct khusus input jurnal
type CreateJournalInput struct {
	Vitals struct {
		Tensi string `json:"tensi"` // "120/80"
		Suhu  string `json:"suhu"`  // "36.5"
		Nadi  string `json:"nadi"`  // "80"
	} `json:"vitals"`
	Notes    string `json:"notes"`
	PhotoURL string `json:"photo_url"`
}

func SubmitMedicalJournal(c *gin.Context) {
	// mitraID, _ := c.Get("userID") // Bisa dipakai untuk validasi hak akses
	orderID := c.Param("id")

	var input CreateJournalInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Input tidak valid", nil)
		return
	}

	// Konversi struct Vitals ke JSON string untuk disimpan di DB
	vitalsJSON, _ := json.Marshal(input.Vitals)

	journal := models.CareJournal{
		OrderID:    utils.StringToUint64(orderID), // Kita butuh helper convert string to uint64
		VitalsData: vitalsJSON,                    // Disimpan sebagai JSON/Blob
		Notes:      input.Notes,
		PhotoURL:   input.PhotoURL,
	}

	if err := config.DB.Create(&journal).Error; err != nil {
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal simpan jurnal", nil)
		return
	}

	// Opsional: Kalau jurnal diisi, apakah status order otomatis COMPLETED?
	// Atau COMPLETED harus tombol terpisah?
	// Untuk MVP, anggap saja kalau sudah isi jurnal berarti kunjungan selesai.
	var order models.Order
	config.DB.First(&order, orderID)
	order.Status = "COMPLETED"
	config.DB.Save(&order)

	utils.APIResponse(c, http.StatusOK, true, "Laporan Medis Tersimpan & Order Selesai", journal)
}
