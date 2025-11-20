package routes

import (
	"homecare-backend/internal/handlers"
	"homecare-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// Grouping API dengan Versi (v1)
	api := r.Group("/api/v1")
	{
		// Grouping Auth
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
		}

		// Route Layanan (Bisa diakses publik biar orang bisa liat harga dulu)
		api.GET("/services", handlers.GetServices)
		api.POST("/payment/notification", handlers.HandleMidtransNotification)

		// 2. PROTECTED ROUTES (Harus Login / Punya Token)
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware()) // <--- PASANG SATPAM DISINI
		{
			// Semua route di dalam kurung kurawal ini otomatis terjaga
			protected.GET("/profile", handlers.GetUserProfile)
			// MODULE PASIEN
			protected.POST("/patients", handlers.AddPatient)
			protected.GET("/patients", handlers.GetMyPatients)

			// MODULE ORDER
			protected.POST("/orders", handlers.CreateOrder)
			protected.GET("/orders", handlers.GetMyOrders)

			// Group Khusus Mitra
			partner := protected.Group("/partner")
			{
				partner.PUT("/profile", handlers.UpdatePartnerProfile)
			}
		}

	}
}
