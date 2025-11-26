package middleware

import (
	"homecare-backend/pkg/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Ambil Header Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.APIResponse(c, http.StatusUnauthorized, false, "Token tidak ditemukan", nil)
			c.Abort()
			return
		}

		// 2. Format harus "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.APIResponse(c, http.StatusUnauthorized, false, "Format token salah", nil)
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 3. Validasi Token
		token, err := utils.ValidateToken(tokenString)
		if err != nil || !token.Valid {
			utils.APIResponse(c, http.StatusUnauthorized, false, "Token tidak valid", nil)
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			utils.APIResponse(c, http.StatusUnauthorized, false, "Gagal memproses token", nil)
			c.Abort()
			return
		}

		// AMAN: JWT Parse number as float64 -> Convert to uint -> Save to Context
		var userID uint64
		if val, ok := claims["user_id"].(float64); ok {
			userID = uint64(val)
		}

		var roleID uint
		if val, ok := claims["role_id"].(float64); ok {
			roleID = uint(val)
		}

		c.Set("userID", userID)
		c.Set("roleID", roleID) // Disimpan sebagai UINT

		c.Next()
	}
}

// AdminOnly: Hanya untuk Role ID 1
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleID, exists := c.Get("roleID")
		if !exists {
			utils.APIResponse(c, http.StatusForbidden, false, "Akses Ditolak", nil)
			c.Abort()
			return
		}

		// PERBAIKAN DI SINI:
		// Karena di AuthMiddleware sudah disimpan sebagai UINT,
		// Maka di sini kita ambil langsung sebagai UINT juga.
		role := roleID.(uint)

		if role != 1 {
			utils.APIResponse(c, http.StatusForbidden, false, "Akses Ditolak: Khusus Admin", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

// FinanceOnly: Hanya untuk Role ID 2 (Atau Admin boleh intip)
func FinanceOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleID, exists := c.Get("roleID")
		if !exists {
			utils.APIResponse(c, http.StatusForbidden, false, "Akses Ditolak", nil)
			c.Abort()
			return
		}

		// PERBAIKAN DI SINI JUGA:
		role := roleID.(uint)

		// Admin (1) juga boleh akses menu finance
		if role != 1 && role != 2 {
			utils.APIResponse(c, http.StatusForbidden, false, "Akses Ditolak: Khusus Finance", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
