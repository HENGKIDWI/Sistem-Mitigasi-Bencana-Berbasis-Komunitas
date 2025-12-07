// cmd/api-kota/main.go
package main

import (
	"log"
	"os"

	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/database"
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/handlers"
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables (misal: .env.kota)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database
	database.ConnectDB()
	defer database.CloseDB()

	// -----------------------------------------------------------------
	// INI ADALAH PERBEDAAN UTAMA:
	// Memanggil migrasi SPESIFIK untuk KOTA
	database.AutoMigrateKota()
	// -----------------------------------------------------------------

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Sistem Mitigasi Bencana (API Kota) v1.0",
		ErrorHandler: customErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: os.Getenv("ALLOWED_ORIGINS"), // Membaca dari .env
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE",
	}))

	// Setup routes
	setupRoutesKota(app)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "4000" // Port default untuk API Kota
	}

	log.Printf("🚀 API Kota (Pusat) berjalan pada port %s", port)
	log.Fatal(app.Listen(":" + port))
}

func setupRoutesKota(app *fiber.App) {
	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Sistem Mitigasi Bencana API (Kota)",
		})
	})

	// API v1
	api := app.Group("/api/v1")

	// Auth routes (Asumsi: menggunakan handler terpisah untuk AdminKota)
	auth := api.Group("/auth")
	// TODO: Anda perlu membuat handler LoginKota yang memeriksa tabel AdminKota
	// Untuk saat ini, kita gunakan handler Login yang ada
	auth.Post("/login", handlers.Login)
	// auth.Post("/register", handlers.Register) // Mungkin tidak perlu register publik

	// Master Data (Sesuai README)
	// TODO: Anda perlu membuat handler untuk ini di handler_kota.go
	master := api.Group("/kecamatan", middleware.AuthMiddleware)
	master.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"message": "Handler GetListKecamatan belum diimplementasi",
		})
	})
	master.Post("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotImplemented).JSON(fiber.Map{
			"message": "Handler CreateKecamatan belum diimplementasi",
		})
	})

	// Monitoring routes (Sesuai README dan kode Anda)
	// monitoring := api.Group("/monitoring", middleware.AuthMiddleware)

	// Hapus middleware.AuthMiddleware agar rute ini jadi PUBLIK buat testing
	monitoring := api.Group("/monitoring")

	monitoring.Get("/kota", handlers.GetMonitoringKota)
	monitoring.Get("/kecamatan/:id", handlers.GetMonitoringKecamatan)

	// 'handlers.GetRekapWilayah' membaca dari tabel RekapDataWilayah
	// yang diisi oleh Sync Worker, jadi ini sudah benar.
	monitoring.Get("/statistik", handlers.GetRekapWilayah) // Sesuai README

	// Reports routes (Sesuai README)
	reports := api.Group("/reports", middleware.AuthMiddleware)
	reports.Get("/dashboard", handlers.GetMonitoringKota) // Re-use handler
	reports.Get("/rekap", handlers.GetRekapWilayah)       // Re-use handler

	// System logs (Hanya untuk admin kota)
	logs := api.Group("/logs", middleware.AuthMiddleware)
	logs.Get("/", handlers.GetSystemLogs)

	// !! RUTE KECAMATAN (Warga, Bencana, Evakuasi) DIHAPUS DARI SINI !!
}

// Error handler kustom
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(fiber.Map{
		"error":   true,
		"message": message,
	})
}
