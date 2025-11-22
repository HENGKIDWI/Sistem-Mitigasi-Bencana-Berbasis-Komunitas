// cmd/api-kecamatan/main.go
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
	// Load environment variables (misal: .env.kecamatan)
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize database
	database.ConnectDB()
	defer database.CloseDB()

	// -----------------------------------------------------------------
	// INI ADALAH PERBEDAAN UTAMA:
	// Memanggil migrasi SPESIFIK untuk KECAMATAN
	database.AutoMigrateKecamatan()
	// -----------------------------------------------------------------

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Sistem Mitigasi Bencana (API Kecamatan) v1.0",
		ErrorHandler: customErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: os.Getenv("ALLOWED_ORIGINS"), // Membaca dari .env
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, PUT, DELETE, PATCH",
	}))

	// Setup routes
	setupRoutes(app)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001" // Port default untuk API Kecamatan
	}

	log.Printf("🚀 API Kecamatan berjalan pada port %s", port)
	log.Fatal(app.Listen(":" + port))
}

func setupRoutes(app *fiber.App) {
	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"message": "Sistem Mitigasi Bencana API (Kecamatan)",
		})
	})

	// API v1
	api := app.Group("/api/v1")

	// Auth routes
	api.Get("/summary", handlers.GetKecamatanSummary)
	auth := api.Group("/auth")
	auth.Post("/login", handlers.Login)
	auth.Post("/register", handlers.Register)
	auth.Get("/me", middleware.AuthMiddleware, handlers.GetProfile)

	// Warga routes
	warga := api.Group("/warga", middleware.AuthMiddleware)
	warga.Get("/", middleware.RoleMiddleware([]string{"RT", "RW", "Admin_Kecamatan"}), handlers.GetAllWarga)
	warga.Get("/:id", handlers.GetWargaByID)
	warga.Post("/", middleware.RoleMiddleware([]string{"RT", "RW"}), handlers.CreateWarga)
	warga.Put("/:id", middleware.RoleMiddleware([]string{"RT", "RW"}), handlers.UpdateWarga)
	warga.Delete("/:id", middleware.RoleMiddleware([]string{"RT", "RW"}), handlers.DeleteWarga)

	// Kejadian Bencana routes
	bencana := api.Group("/bencana", middleware.AuthMiddleware)
	bencana.Get("/", handlers.GetAllBencana)
	bencana.Get("/:id", handlers.GetBencanaByID)
	bencana.Post("/", middleware.RoleMiddleware([]string{"RT", "RW", "Admin_Kecamatan"}), handlers.CreateBencana)
	bencana.Put("/:id/status", handlers.UpdateStatusBencana)
	bencana.Get("/active", handlers.GetActiveBencana)

	// Evakuasi routes
	evakuasi := api.Group("/evakuasi", middleware.AuthMiddleware)
	evakuasi.Get("/prioritas/:bencana_id", handlers.GetPrioritasEvakuasi)
	evakuasi.Post("/log", middleware.RoleMiddleware([]string{"Relawan"}), handlers.CreateLogEvakuasi)
	evakuasi.Put("/log/:id", middleware.RoleMiddleware([]string{"Relawan"}), handlers.UpdateLogEvakuasi)
	evakuasi.Get("/log/:bencana_id", handlers.GetLogEvakuasi)

	// Notifikasi & Broadcast routes
	notif := api.Group("/notifikasi", middleware.AuthMiddleware)
	notif.Post("/darurat", middleware.RoleMiddleware([]string{"RT", "RW", "Admin_Kecamatan"}), handlers.SendDaruratNotification)

	// SSE untuk broadcast
	// Ini tetap di sini agar RT/Relawan bisa mendapat update real-time
	app.Get("/api/v1/broadcast/stream", handlers.BroadcastStream)

	// System logs (Log lokal untuk kecamatan ini)
	logs := api.Group("/logs", middleware.AuthMiddleware, middleware.RoleMiddleware([]string{"Admin_Kecamatan"}))
	logs.Get("/", handlers.GetSystemLogs)

	//

	// !! RUTE KOTA (monitoring/kota, monitoring/kecamatan, rekap) DIHAPUS DARI SINI !!
}

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
