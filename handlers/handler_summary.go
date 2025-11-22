// handlers/handler_summary.go
package handlers

import (
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/database"
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/models"
	"github.com/gofiber/fiber/v2"
)

// GetKecamatanSummary adalah endpoint yang akan dipanggil oleh Sync Worker
func GetKecamatanSummary(c *fiber.Ctx) error {
	var totalWarga int64
	var totalRentan int64
	var bencanaAktif int64

	// 1. Hitung total warga
	if err := database.DB.Model(&models.WargaRentan{}).Count(&totalWarga).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Gagal menghitung total warga",
		})
	}

	// 2. Hitung total kelompok rentan
	if err := database.DB.Model(&models.WargaRentan{}).Where("kategori_rentan != ?", "Non-Rentan").Count(&totalRentan).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Gagal menghitung total rentan",
		})
	}

	// 3. Hitung bencana aktif
	if err := database.DB.Model(&models.KejadianBencana{}).Where("status = ?", "Aktif").Count(&bencanaAktif).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Gagal menghitung bencana aktif",
		})
	}

	// Kembalikan data ringkasan
	return c.JSON(fiber.Map{
		"error": false,
		"data": fiber.Map{
			"total_warga":   totalWarga,
			"total_rentan":  totalRentan,
			"bencana_aktif": bencanaAktif,
			// Anda bisa tambahkan 'kecamatan_id' dari .env jika perlu
		},
	})
}
