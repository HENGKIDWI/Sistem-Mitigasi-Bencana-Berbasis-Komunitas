// handlers/warga.go
package handlers

import (
	"strconv"

	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/database"
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/models"
	"github.com/gofiber/fiber/v2"
)

// GetAllWarga returns all warga rentan with optional filters
func GetAllWarga(c *fiber.Ctx) error {
	var warga []models.WargaRentan

	query := database.DB

	// Filter by RT
	if rt := c.Query("rt"); rt != "" {
		query = query.Where("rt = ?", rt)
	}

	// Filter by RW
	if rw := c.Query("rw"); rw != "" {
		query = query.Where("rw = ?", rw)
	}

	// Filter by kategori
	if kategori := c.Query("kategori_rentan"); kategori != "" {
		query = query.Where("kategori_rentan = ?", kategori)
	}

	// Execute query
	if err := query.Order("skor_prioritas DESC").Find(&warga).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to fetch warga data",
		})
	}

	return c.JSON(fiber.Map{
		"error": false,
		"data":  warga,
		"total": len(warga),
	})
}

// GetWargaByID returns single warga by ID
func GetWargaByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid ID",
		})
	}

	var warga models.WargaRentan
	if err := database.DB.First(&warga, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "Warga not found",
		})
	}

	return c.JSON(fiber.Map{
		"error": false,
		"data":  warga,
	})
}

// CreateWarga creates new warga rentan
func CreateWarga(c *fiber.Ctx) error {
	var req models.CreateWargaRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid request body",
		})
	}

	// Calculate priority score based on category
	skorPrioritas := calculatePriorityScore(req.KategoriRentan)

	warga := models.WargaRentan{
		NIK:            req.NIK,
		Nama:           req.Nama,
		Alamat:         req.Alamat,
		RT:             req.RT,
		RW:             req.RW,
		KategoriRentan: req.KategoriRentan,
		SkorPrioritas:  skorPrioritas,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		NoHP:           req.NoHP,
	}

	if err := database.DB.Create(&warga).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to create warga",
		})
	}

	// Log activity
	userID := c.Locals("userID").(uint)
	logActivity(userID, "Menambahkan warga rentan: "+warga.Nama)

	// -----------------------------------------------------------------
	// DIHAPUS: go updateRekapWilayah(req.RT, req.RW)
	// Tugas ini sekarang dilakukan oleh Sync Worker
	// -----------------------------------------------------------------

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"error":   false,
		"message": "Warga created successfully",
		"data":    warga,
	})
}

// UpdateWarga updates existing warga
func UpdateWarga(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid ID",
		})
	}

	var warga models.WargaRentan
	if err := database.DB.First(&warga, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "Warga not found",
		})
	}

	var req models.CreateWargaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid request body",
		})
	}

	// Update fields
	warga.NIK = req.NIK
	warga.Nama = req.Nama
	warga.Alamat = req.Alamat
	warga.RT = req.RT
	warga.RW = req.RW
	warga.KategoriRentan = req.KategoriRentan
	warga.SkorPrioritas = calculatePriorityScore(req.KategoriRentan)
	warga.Latitude = req.Latitude
	warga.Longitude = req.Longitude
	warga.NoHP = req.NoHP

	if err := database.DB.Save(&warga).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to update warga",
		})
	}

	// Log activity
	userID := c.Locals("userID").(uint)
	logActivity(userID, "Mengupdate warga rentan: "+warga.Nama)

	return c.JSON(fiber.Map{
		"error":   false,
		"message": "Warga updated successfully",
		"data":    warga,
	})
}

// DeleteWarga soft deletes warga
func DeleteWarga(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid ID",
		})
	}

	var warga models.WargaRentan
	if err := database.DB.First(&warga, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "Warga not found",
		})
	}

	if err := database.DB.Delete(&warga).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to delete warga",
		})
	}

	// Log activity
	userID := c.Locals("userID").(uint)
	logActivity(userID, "Menghapus warga rentan: "+warga.Nama)

	return c.JSON(fiber.Map{
		"error":   false,
		"message": "Warga deleted successfully",
	})
}

// Helper function to calculate priority score
func calculatePriorityScore(kategori string) int {
	scores := map[string]int{
		"Disabilitas": 100,
		"Sakit Keras": 95,
		"Lansia":      90,
		"Ibu Hamil":   85,
		"Anak-anak":   80,
		"Non-Rentan":  50,
	}

	if score, exists := scores[kategori]; exists {
		return score
	}
	return 50
}

// Helper function to update rekap wilayah (async)
// DIHAPUS - Fungsinya dipindahkan ke Sync Worker
// func updateRekapWilayah(rt, rw string) {
// }
