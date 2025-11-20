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
			c.Abort() // Stop, jangan lanjut ke handler
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
			utils.APIResponse(c, http.StatusUnauthorized, false, "Token tidak valid atau kadaluarsa", nil)
			c.Abort()
			return
		}

		// 4. Ambil Data Claims (User ID & Role) dari Token
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			utils.APIResponse(c, http.StatusUnauthorized, false, "Gagal memproses token", nil)
			c.Abort()
			return
		}

		// 5. Simpan User ID ke Context Gin (Biar bisa dipake di Controller nanti)
		// Note: JSON number biasanya jadi float64 di Go
		userID := uint64(claims["user_id"].(float64))
		roleID := uint(claims["role_id"].(float64))

		c.Set("userID", userID)
		c.Set("roleID", roleID)

		c.Next() // Lanjut ke controller tujuan
	}
}
