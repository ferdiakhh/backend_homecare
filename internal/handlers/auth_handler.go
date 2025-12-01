package handlers

import (
	"homecare-backend/internal/config"
	"homecare-backend/internal/models"
	"homecare-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// REGISTER
func Register(c *gin.Context) {
	var input models.RegisterInput

	// 1. Validasi Input JSON
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Input tidak valid", err.Error())
		return
	}

	// 2. Cek Password Hash
	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal memproses password", nil)
		return
	}

	// 3. Siapkan Data User
	user := models.User{
		FullName:     input.FullName,
		Email:        input.Email,
		PasswordHash: hashedPassword,
		RoleID:       input.RoleID,
		Phone:        input.Phone,
		IsVerified:   false, // Default belum verifikasi email
	}

	// 4. Simpan ke Database
	// Note: Role ID 3=Mitra, 4=Customer. Pastikan Role ID valid.
	if err := config.DB.Create(&user).Error; err != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Email atau Nomor HP sudah terdaftar!", nil)
		return
	}

	// 5. Sukses
	utils.APIResponse(c, http.StatusCreated, true, "Registrasi Berhasil! Silakan Login.", user)
}

// LOGIN
func Login(c *gin.Context) {
	var input models.LoginInput

	// 1. Validasi Input
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.APIResponse(c, http.StatusBadRequest, false, "Input tidak valid", nil)
		return
	}

	// 2. Cari User berdasarkan Email
	var user models.User
	if err := config.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		utils.APIResponse(c, http.StatusUnauthorized, false, "Email atau Password salah", nil)
		return
	}

	// 3. Cek Password
	if !utils.CheckPassword(input.Password, user.PasswordHash) {
		utils.APIResponse(c, http.StatusUnauthorized, false, "Email atau Password salah", nil)
		return
	}

	// ===> LOGIKA UPDATE FCM TOKEN (BARU) <===
	// Jika frontend mengirim token FCM, simpan ke database
	if input.FCMToken != "" {
		user.FCMToken = input.FCMToken
		// Kita hanya update kolom fcm_token agar efisien
		config.DB.Model(&user).Update("fcm_token", input.FCMToken)
	}

	// 4. Generate JWT Token
	token, err := utils.GenerateToken(user.ID, user.RoleID)
	if err != nil {
		utils.APIResponse(c, http.StatusInternalServerError, false, "Gagal generate token", nil)
		return
	}

	// 5. Sukses & Kirim Token
	utils.APIResponse(c, http.StatusOK, true, "Login Berhasil", gin.H{
		"token": token,
		"user": gin.H{
			"id":        user.ID,
			"full_name": user.FullName,
			"role_id":   user.RoleID,
			"email":     user.Email,
		},
	})
}
