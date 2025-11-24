package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// IPRateLimiter menyimpan daftar limiter untuk setiap IP
type IPRateLimiter struct {
	ips map[string]*visitor
	mu  *sync.RWMutex
	r   rate.Limit // Rate: berapa request per detik
	b   int        // Burst: toleransi lonjakan sesaat
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewIPRateLimiter membuat instance limiter baru
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	i := &IPRateLimiter{
		ips: make(map[string]*visitor),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}

	// Jalankan "Tukang Sampah" (Cleanup) di background setiap 1 menit
	// Untuk menghapus IP yang sudah lama tidak aktif agar hemat RAM
	go i.cleanupVisitors()

	return i
}

// GetLimiter mengambil/membuat limiter untuk IP tertentu
func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	v, exists := i.ips[ip]
	if !exists {
		// Kalau IP baru, buatkan limiter baru
		limiter := rate.NewLimiter(i.r, i.b)
		i.ips[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	// Update waktu terakhir akses
	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupVisitors menghapus IP yang sudah 3 menit tidak aktif
func (i *IPRateLimiter) cleanupVisitors() {
	for {
		time.Sleep(1 * time.Minute)
		i.mu.Lock()
		for ip, v := range i.ips {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(i.ips, ip)
			}
		}
		i.mu.Unlock()
	}
}

// Middleware Function yang akan dipanggil di route
func RateLimitMiddleware() gin.HandlerFunc {
	// KONFIGURASI:
	// 5 request per detik, dengan toleransi lonjakan (burst) sampai 10 request.
	// Ini cukup longgar untuk user normal, tapi cukup ketat buat bot.
	limiter := NewIPRateLimiter(5, 10)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		if limiter := limiter.GetLimiter(ip); !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"message": "Terlalu banyak request! Santai dulu kawan.",
			})
			return
		}
		c.Next()
	}
}
