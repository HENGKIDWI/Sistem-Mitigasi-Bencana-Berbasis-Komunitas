// handlers/logistik.go
package handlers

import (
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/database"
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/models"
	"github.com/gofiber/fiber/v2"
)

// GetAllLogistik - Melihat semua stok barang
func GetAllLogistik(c *fiber.Ctx) error {
	var barang []models.Logistik

	query := database.DB
	if kategori := c.Query("kategori"); kategori != "" {
		query = query.Where("kategori = ?", kategori)
	}

	if err := query.Find(&barang).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": true, "message": "Gagal mengambil data logistik",
		})
	}

	return c.JSON(fiber.Map{"error": false, "data": barang})
}

// CreateLogistik - Mendaftarkan barang baru (Master Data)
func CreateLogistik(c *fiber.Ctx) error {
	var req models.CreateLogistikRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": true, "message": "Invalid body"})
	}

	barang := models.Logistik{
		Nama: req.Nama, Satuan: req.Satuan, Kategori: req.Kategori,
		Lokasi: req.Lokasi, Stok: req.StokAwal,
	}

	if err := database.DB.Create(&barang).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": true, "message": "Gagal membuat barang"})
	}

	// Catat log aktivitas
	userID := c.Locals("userID").(uint)
	logActivity(userID, "Menambahkan item logistik baru: "+barang.Nama)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"error": false, "data": barang})
}

// HandleTransaksiLogistik - Menangani Barang Masuk/Keluar
func HandleTransaksiLogistik(c *fiber.Ctx) error {
	var req models.TransaksiLogistikRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": true, "message": "Invalid body"})
	}

	userID := c.Locals("userID").(uint)

	// Mulai Database Transaction (Penting untuk konsistensi stok)
	tx := database.DB.Begin()

	// 1. Cek Barang ada atau tidak
	var barang models.Logistik
	if err := tx.First(&barang, req.LogistikID).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": true, "message": "Barang tidak ditemukan"})
	}

	// 2. Update Stok berdasarkan jenis transaksi
	if req.Jenis == "Masuk" {
		barang.Stok += req.Jumlah
	} else if req.Jenis == "Keluar" {
		if barang.Stok < req.Jumlah {
			tx.Rollback()
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": true, "message": "Stok tidak mencukupi"})
		}
		barang.Stok -= req.Jumlah
	} else {
		tx.Rollback()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": true, "message": "Jenis transaksi harus Masuk/Keluar"})
	}

	// 3. Simpan Perubahan Stok
	if err := tx.Save(&barang).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": true, "message": "Gagal update stok"})
	}

	// 4. Catat Riwayat Transaksi
	transaksi := models.LogistikTransaksi{
		LogistikID: req.LogistikID,
		UserID:     userID,
		Jenis:      req.Jenis,
		Jumlah:     req.Jumlah,
		Keterangan: req.Keterangan,
	}

	// Jika ada ID bencana (opsional)
	if req.BencanaID != 0 {
		transaksi.BencanaID = &req.BencanaID
	}

	if err := tx.Create(&transaksi).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": true, "message": "Gagal mencatat transaksi"})
	}

	// Commit Transaction
	tx.Commit()

	// Log System
	logActivity(userID, "Transaksi Logistik "+req.Jenis+": "+barang.Nama)

	return c.JSON(fiber.Map{
		"error":     false,
		"message":   "Transaksi berhasil",
		"sisa_stok": barang.Stok,
	})
}

// GetRiwayatLogistik - Melihat history keluar masuk barang
func GetRiwayatLogistik(c *fiber.Ctx) error {
	id := c.Params("id") // ID Barang
	var riwayat []models.LogistikTransaksi

	if err := database.DB.Where("logistik_id = ?", id).Preload("User").Order("created_at DESC").Find(&riwayat).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": true, "message": "Gagal ambil riwayat"})
	}

	return c.JSON(fiber.Map{"error": false, "data": riwayat})
}
