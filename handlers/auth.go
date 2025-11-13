// handlers/auth.go
package handlers

import (
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/database"
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/middleware"
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/models"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// Login handler
func Login(c *fiber.Ctx) error {
	var req models.LoginRequest

	// Parse request body
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid request body",
		})
	}

	// Find user by username
	var user models.User
	if err := database.DB.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid credentials",
		})
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid credentials",
		})
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(user.ID, user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to generate token",
		})
	}

	// Log login activity
	logActivity(user.ID, "User login")

	return c.JSON(fiber.Map{
		"error": false,
		"data": models.LoginResponse{
			Token: token,
			User:  user,
		},
	})
}

// Register handler
// handlers/auth.go

// Register handler
func Register(c *fiber.Ctx) error {
	var req models.RegisterRequest // <-- UBAH INI (1)

	// Parse request body
	if err := c.BodyParser(&req); err != nil { // <-- UBAH INI (2)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid request body",
		})
	}

	// Validate required fields
	// Validasi dari 'req', bukan 'user'
	if req.Username == "" || req.Password == "" || req.Role == "" || req.NamaLengkap == "" { // <-- UBAH INI (3)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Missing required fields",
		})
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost) // <-- UBAH INI (4)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to hash password",
		})
	}

	// Buat models.User dari req
	user := models.User{
		Username:     req.Username,
		Password:     string(hashedPassword), // <-- Gunakan password yang sudah di-hash
		Role:         req.Role,
		NamaLengkap:  req.NamaLengkap,
		NoHP:         req.NoHP,
		WilayahTugas: req.WilayahTugas,
	} // <-- TAMBAHKAN BLOK INI (5)

	// Create user
	if err := database.DB.Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to create user",
		})
	}

	// Remove password from response
	user.Password = ""

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"error":   false,
		"message": "User created successfully",
		"data":    user,
	})
}

// GetProfile handler
func GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("userID").(uint)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "User not found",
		})
	}

	return c.JSON(fiber.Map{
		"error": false,
		"data":  user,
	})
}

// Helper function to log activity
func logActivity(userID uint, activity string) {
	log := models.SystemLog{
		UserID:    userID,
		Aktivitas: activity,
	}
	database.DB.Create(&log)
}
