// handlers/bencana.go
package handlers

import (
	"strconv"
	"time"

	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/database"
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/models"
	"github.com/gofiber/fiber/v2"
)

// GetAllBencana returns all kejadian bencana
func GetAllBencana(c *fiber.Ctx) error {
	var bencana []models.KejadianBencana

	query := database.DB.Preload("UserPelapor")

	// Filter by status
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// Filter by level
	if level := c.Query("level"); level != "" {
		query = query.Where("level = ?", level)
	}

	if err := query.Order("waktu_mulai DESC").Find(&bencana).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to fetch bencana data",
		})
	}

	return c.JSON(fiber.Map{
		"error": false,
		"data":  bencana,
		"total": len(bencana),
	})
}

// GetBencanaByID returns single bencana by ID
func GetBencanaByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid ID",
		})
	}

	var bencana models.KejadianBencana
	if err := database.DB.Preload("UserPelapor").First(&bencana, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "Bencana not found",
		})
	}

	return c.JSON(fiber.Map{
		"error": false,
		"data":  bencana,
	})
}

// GetActiveBencana returns all active bencana
func GetActiveBencana(c *fiber.Ctx) error {
	var bencana []models.KejadianBencana

	if err := database.DB.Where("status = ?", "Aktif").
		Preload("UserPelapor").
		Order("waktu_mulai DESC").
		Find(&bencana).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to fetch active bencana",
		})
	}

	return c.JSON(fiber.Map{
		"error": false,
		"data":  bencana,
		"total": len(bencana),
	})
}

// CreateBencana creates new kejadian bencana
func CreateBencana(c *fiber.Ctx) error {
	var req models.CreateBencanaRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid request body",
		})
	}

	userID := c.Locals("userID").(uint)

	bencana := models.KejadianBencana{
		JenisBencana:  req.JenisBencana,
		Level:         req.Level,
		WaktuMulai:    time.Now(),
		Status:        "Aktif",
		UserPelaporID: userID,
		Deskripsi:     req.Deskripsi,
	}

	if err := database.DB.Create(&bencana).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to create bencana",
		})
	}

	// Preload user pelapor
	database.DB.Preload("UserPelapor").First(&bencana, bencana.ID)

	// Log activity
	logActivity(userID, "Melaporkan bencana: "+bencana.JenisBencana)

	// Trigger notification to relevant parties
	go triggerBencanaNotification(bencana)

	// Create monitoring entry
	go createMonitoringBencana(bencana)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"error":   false,
		"message": "Bencana reported successfully",
		"data":    bencana,
	})
}

// UpdateStatusBencana updates status of bencana
func UpdateStatusBencana(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid ID",
		})
	}

	var bencana models.KejadianBencana
	if err := database.DB.First(&bencana, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "Bencana not found",
		})
	}

	var req struct {
		Status string `json:"status"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid request body",
		})
	}

	bencana.Status = req.Status
	if req.Status == "Selesai" {
		now := time.Now()
		bencana.WaktuSelesai = &now
	}

	if err := database.DB.Save(&bencana).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to update bencana status",
		})
	}

	// Log activity
	userID := c.Locals("userID").(uint)
	logActivity(userID, "Mengubah status bencana menjadi: "+req.Status)

	return c.JSON(fiber.Map{
		"error":   false,
		"message": "Bencana status updated successfully",
		"data":    bencana,
	})
}

// GetPrioritasEvakuasi returns prioritized evacuation list
func GetPrioritasEvakuasi(c *fiber.Ctx) error {
	bencanaID, err := strconv.Atoi(c.Params("bencana_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid bencana ID",
		})
	}

	// Get bencana info
	var bencana models.KejadianBencana
	if err := database.DB.First(&bencana, bencanaID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "Bencana not found",
		})
	}

	// Get warga with priority score
	var warga []models.WargaRentan
	query := database.DB.Where("kategori_rentan != ?", "Non-Rentan")

	// Filter based on bencana level
	if bencana.Level == "Lokal_RT" {
		// Get RT from user who reported
		var user models.User
		database.DB.First(&user, bencana.UserPelaporID)
		// Assuming wilayah_tugas format: "RT 001/RW 001"
		// This needs to be parsed properly
		query = query.Where("rt = ?", extractRT(user.WilayahTugas))
	}

	if err := query.Order("skor_prioritas DESC, nama ASC").Find(&warga).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to fetch evacuation priority",
		})
	}

	return c.JSON(fiber.Map{
		"error":     false,
		"bencana":   bencana,
		"prioritas": warga,
		"total":     len(warga),
	})
}

// CreateLogEvakuasi creates evacuation log
func CreateLogEvakuasi(c *fiber.Ctx) error {
	var log models.LogEvakuasi

	if err := c.BodyParser(&log); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid request body",
		})
	}

	log.RelawanID = c.Locals("userID").(uint)
	log.WaktuUpdate = time.Now()

	if err := database.DB.Create(&log).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to create evacuation log",
		})
	}

	// Preload relations
	database.DB.Preload("Bencana").Preload("Warga").Preload("Relawan").First(&log, log.ID)

	// Log activity
	logActivity(log.RelawanID, "Update status evakuasi warga")

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"error":   false,
		"message": "Evacuation log created successfully",
		"data":    log,
	})
}

// UpdateLogEvakuasi updates evacuation log
func UpdateLogEvakuasi(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid ID",
		})
	}

	var log models.LogEvakuasi
	if err := database.DB.First(&log, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "Log not found",
		})
	}

	var req struct {
		StatusTerkini string `json:"status_terkini"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid request body",
		})
	}

	log.StatusTerkini = req.StatusTerkini
	log.WaktuUpdate = time.Now()

	if err := database.DB.Save(&log).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to update evacuation log",
		})
	}

	// Log activity
	userID := c.Locals("userID").(uint)
	logActivity(userID, "Update status evakuasi")

	return c.JSON(fiber.Map{
		"error":   false,
		"message": "Evacuation log updated successfully",
		"data":    log,
	})
}

// GetLogEvakuasi returns all evacuation logs for a bencana
func GetLogEvakuasi(c *fiber.Ctx) error {
	bencanaID, err := strconv.Atoi(c.Params("bencana_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid bencana ID",
		})
	}

	var logs []models.LogEvakuasi
	if err := database.DB.Where("bencana_id = ?", bencanaID).
		Preload("Warga").
		Preload("Relawan").
		Order("waktu_update DESC").
		Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to fetch evacuation logs",
		})
	}

	return c.JSON(fiber.Map{
		"error": false,
		"data":  logs,
		"total": len(logs),
	})
}

// Helper functions
func triggerBencanaNotification(bencana models.KejadianBencana) {
	// Implementation for sending notifications via WhatsApp or other channels
	// This would integrate with notification service
}

func createMonitoringBencana(bencana models.KejadianBencana) {
	// Create monitoring entry at city level
	// This would sync data to MonitoringBencanaKota table
}

func extractRT(wilayahTugas string) string {
	// Parse RT from wilayah_tugas string
	// Example: "RT 001/RW 001" -> "001"
	// Implement proper parsing logic
	return ""
}
