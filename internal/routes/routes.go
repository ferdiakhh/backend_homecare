package routes

import (
	"homecare-backend/internal/handlers"
	"homecare-backend/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {

	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RateLimitMiddleware())
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
		api.GET("/partners/search", handlers.SearchPartners)

		// 2. PROTECTED ROUTES (Harus Login / Punya Token)
		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware()) // <--- PASANG SATPAM DISINI
		{
			// Semua route di dalam kurung kurawal ini otomatis terjaga
			protected.GET("/profile", handlers.GetUserProfile)
			// MODULE PASIEN
			protected.POST("/patients", handlers.AddPatient)
			protected.GET("/patients", handlers.GetMyPatients)
			protected.GET("/patients/:id/history", handlers.GetPatientHistory)

			// MODULE ORDER
			protected.POST("/orders", handlers.CreateOrder)
			protected.GET("/orders", handlers.GetMyOrders)
			protected.GET("/orders/:id", handlers.GetOrderDetail)

			// Group Khusus Mitra
			partner := protected.Group("/partner")
			{
				partner.PUT("/profile", handlers.UpdatePartnerProfile)
				partner.GET("/profile/me", handlers.GetMyPartnerProfile)
				partner.PATCH("/status", handlers.TogglePartnerStatus)
				partner.GET("/orders/my-jobs", handlers.GetMyJobs)
				// 1. Liat Job
				partner.GET("/orders/available", handlers.GetAvailableOrders)

				// 2. Ambil Job
				partner.POST("/orders/:id/accept", handlers.AcceptOrder)
				partner.POST("/orders/:id/reject", handlers.RejectOrder)

				// 3. Lapor Kerja (Jurnal)
				partner.POST("/orders/:id/journal", handlers.SubmitMedicalJournal)

				// MODUL KEUANGAN
				partner.GET("/wallet", handlers.GetMyWallet)
				partner.POST("/wallet/withdraw", handlers.RequestWithdrawal)
			}

			// Group ADMIN
			admin := protected.Group("/admin")
			{
				// Dashboard Utama
				admin.GET("/dashboard", middleware.AdminOnly(), handlers.GetDashboardStats)

				// Manajemen Customer
				admin.GET("/customers", middleware.AdminOnly(), handlers.GetAllCustomers)

				// Manajemen Mitra
				admin.GET("/partners", middleware.AdminOnly(), handlers.GetAllPartners)

				// Manajemen Order (Ops)
				admin.GET("/orders", middleware.AdminOnly(), handlers.GetAllOrders)

				// Manajemen Service (Ops)
				admin.PUT("/services/:id", middleware.AdminOnly(), handlers.UpdateServicePrice)
				// Modul Mitra (Ops)
				admin.GET("/partners/pending", middleware.AdminOnly(), handlers.GetPendingPartners)
				admin.POST("/partners/:id/verify", middleware.AdminOnly(), handlers.VerifyPartner)

				// Modul Keuangan (Finance)
				admin.GET("/withdrawals", middleware.FinanceOnly(), handlers.GetAllWithdrawals)
				admin.POST("/withdrawals/:id/process", middleware.FinanceOnly(), handlers.ApproveWithdrawal)
			}
		}

	}
}
