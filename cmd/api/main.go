package main

import (
	"log"
	"os"

	"homecare-backend/internal/config"
	"homecare-backend/internal/middleware"
	"homecare-backend/internal/routes" // <--- Import ini
	"homecare-backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 1. Load Env
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found")
	}

	// 2. Connect DB
	config.ConnectDB()

	// Init Firebase
	utils.InitFCM()

	// 3. Init Router
	r := gin.Default()

	// 4. Pasang Middleware Global
	r.Use(middleware.CORSMiddleware())

	// 5. Setup Routes (Panggil fungsi dari folder routes)
	routes.SetupRoutes(r) // <--- Kodingan routing cuma sebaris ini jadinya

	// 6. Test Ping (Opsional, boleh dihapus atau dipindah ke routes juga)
	r.GET("/ping", func(c *gin.Context) {
		utils.APIResponse(c, 200, true, "Server OK!", nil)
	})

	// 7. Run Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server berjalan di port " + port)
	r.Run(":" + port)
}
