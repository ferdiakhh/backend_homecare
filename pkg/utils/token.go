package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateToken membuat JWT string yang berisi User ID dan Role
func GenerateToken(userID uint64, roleID uint) (string, error) {
	// Kunci Rahasia (Sebaiknya simpan di .env dengan nama JWT_SECRET)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "rahasia_dapur_homecare" // Fallback kalau .env lupa diisi
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"role_id": roleID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Token berlaku 24 jam
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken memverifikasi apakah token valid atau tidak
func ValidateToken(encodedToken string) (*jwt.Token, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "rahasia_dapur_homecare"
	}

	return jwt.Parse(encodedToken, func(token *jwt.Token) (interface{}, error) {
		// Validasi algoritma enkripsi (harus HMAC)
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
}
