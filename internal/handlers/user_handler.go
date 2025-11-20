package handlers

import (
	"homecare-backend/internal/config"
	"homecare-backend/internal/models"
	"homecare-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetUserProfile mengambil data user yang sedang login
func GetUserProfile(c *gin.Context) {
	// 1. Ambil User ID dari Context (Hasil kerja Middleware tadi)
	userID, exists := c.Get("userID")
	if !exists {
		utils.APIResponse(c, http.StatusUnauthorized, false, "Unauthorized", nil)
		return
	}

	// 2. Cari di Database
	var user models.User
	if err := config.DB.First(&user, userID).Error; err != nil {
		utils.APIResponse(c, http.StatusNotFound, false, "User tidak ditemukan", nil)
		return
	}

	// 3. Return Data (Tanpa Password)
	utils.APIResponse(c, http.StatusOK, true, "Data Profile Berhasil Diambil", gin.H{
		"id":        user.ID,
		"full_name": user.FullName,
		"email":     user.Email,
		"phone":     user.Phone,
		"role_id":   user.RoleID,
	})
}
