// handlers/monitoring.go
package handlers

import (
	"bufio"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/database"
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/models"
	"github.com/gofiber/fiber/v2"
)

// SSE Broadcast management
// (Ini mungkin TIDAK akan digunakan oleh API Kota,
// tapi kita biarkan saja. API Kecamatan yang akan banyak menggunakan ini)
var (
	broadcastClients = make(map[string]chan string)
	clientsMutex     sync.RWMutex
)

// GetMonitoringKecamatan returns monitoring data for a kecamatan
// FUNGSI INI HANYA DIJALANKAN DI API KOTA, MEMBACA DB KOTA
func GetMonitoringKecamatan(c *fiber.Ctx) error {
	kecamatanID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid kecamatan ID",
		})
	}

	// REFAKTOR: Ambil bencana dari tabel agregasi
	var bencana []models.MonitoringBencanaKota
	database.DB.Where("kecamatan_id = ?", kecamatanID).
		Preload("Kecamatan").
		Order("waktu_laporan DESC").
		Find(&bencana)

	// REFAKTOR: Ambil rekap dari tabel agregasi
	var rekap models.RekapDataWilayah
	database.DB.Where("kecamatan_id = ?", kecamatanID).
		Preload("Kecamatan").
		First(&rekap)

	// REFAKTOR: Query ke LogEvakuasi DIHAPUS
	// Data ini harusnya sudah ada di tabel rekap/monitoring jika diperlukan,
	// atau butuh endpoint summary baru di API Kecamatan

	return c.JSON(fiber.Map{
		"error": false,
		"data": fiber.Map{
			// Hanya mengembalikan data dari tabel agregasi
			"bencana_monitoring": bencana,
			"rekap_wilayah":      rekap,
		},
	})
}

// GetMonitoringKota returns monitoring data for entire city
// FUNGSI INI HANYA DIJALANKAN DI API KOTA, MEMBACA DB KOTA
func GetMonitoringKota(c *fiber.Ctx) error {
	// BENAR: Ambil data monitoring dari tabel agregasi
	var monitoring []models.MonitoringBencanaKota
	database.DB.Preload("Kecamatan").
		Order("waktu_laporan DESC").
		Limit(50).
		Find(&monitoring)

	// BENAR: Ambil rekap dari tabel agregasi
	var rekapData []models.RekapDataWilayah
	database.DB.Preload("Kecamatan").Find(&rekapData)

	// BENAR: Hitung total statistik dari tabel rekap
	var totalWarga, totalKerentanan int
	for _, rekap := range rekapData {
		totalWarga += rekap.TotalWarga
		totalKerentanan += rekap.TotalKerentanan
	}

	// REFAKTOR: Hitung bencana aktif dari tabel MONITORING, bukan KejadianBencana
	var activeBencanaCount int64
	// Asumsi: Sync worker sudah mengisi 'total_bencana' dengan benar
	database.DB.Model(&models.MonitoringBencanaKota{}).
		Select("SUM(total_bencana)").
		Scan(&activeBencanaCount)

	return c.JSON(fiber.Map{
		"error": false,
		"data": fiber.Map{
			"monitoring":           monitoring,
			"rekap_kecamatan":      rekapData,
			"total_warga":          totalWarga,
			"total_kerentanan":     totalKerentanan,
			"active_bencana_count": activeBencanaCount, // Data sudah benar dari DB Kota
		},
	})
}

// GetRekapWilayah returns rekap data for a specific kecamatan
// FUNGSI INI HANYA DIJALANKAN DI API KOTA, MEMBACA DB KOTA
func GetRekapWilayah(c *fiber.Ctx) error {
	kecamatanID, err := strconv.Atoi(c.Params("kecamatan_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid kecamatan ID",
		})
	}

	// BENAR: Ambil rekap dari tabel agregasi
	var rekap models.RekapDataWilayah
	if err := database.DB.Where("kecamatan_id = ?", kecamatanID).
		Preload("Kecamatan").
		First(&rekap).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "Rekap not found",
		})
	}

	// REFAKTOR: Query breakdown DIHAPUS.
	// Query ini tidak bisa dijalankan di DB Kota karena tidak ada tabel WargaRentan.
	// Data breakdown ini harusnya disiapkan oleh Sync Worker
	// dan disimpan di tabel RekapDataWilayah.

	return c.JSON(fiber.Map{
		"error": false,
		"data": fiber.Map{
			"rekap": rekap,
			// "breakdown" DIHAPUS
		},
	})
}

// SendDaruratNotification sends emergency notification
// FUNGSI INI HANYA DIJALANKAN DI API KECAMATAN
func SendDaruratNotification(c *fiber.Ctx) error {
	var req models.BroadcastRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid request body",
		})
	}

	// Get bencana details
	var bencana models.KejadianBencana
	if err := database.DB.First(&bencana, req.BencanaID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "Bencana not found",
		})
	}

	// Format broadcast message
	message := fmt.Sprintf(
		"{\"jenis\": \"%s\", \"level\": \"%s\", \"pesan\": \"%s\", \"waktu\": \"%s\"}",
		bencana.JenisBencana,
		bencana.Level,
		req.Message,
		time.Now().Format(time.RFC3339),
	)

	// Broadcast to all connected SSE clients
	broadcastToClients(message)

	// Send WhatsApp notifications (integrate with WA API)
	go sendWhatsAppNotifications(bencana, req.Message)

	// Log activity
	userID := c.Locals("userID").(uint)
	logActivity(userID, "Mengirim notifikasi darurat: "+bencana.JenisBencana)

	return c.JSON(fiber.Map{
		"error":   false,
		"message": "Emergency notification sent successfully",
		"data": fiber.Map{
			"broadcast_message": message,
			"timestamp":         time.Now(),
		},
	})
}

// BroadcastStream handles SSE connections
// FUNGSI INI HANYA DIJALANKAN DI API KECAMATAN
func BroadcastStream(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	// Create unique client ID
	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())

	// Create message channel for this client
	messageChan := make(chan string, 10)

	// Register client
	clientsMutex.Lock()
	broadcastClients[clientID] = messageChan
	clientsMutex.Unlock()

	// Remove client on disconnect
	defer func() {
		clientsMutex.Lock()
		delete(broadcastClients, clientID)
		close(messageChan)
		clientsMutex.Unlock()
	}()

	// Send initial connection message
	c.Write([]byte("event: connected\n"))
	c.Write([]byte(fmt.Sprintf("data: {\"clientId\": \"%s\", \"message\": \"Connected to broadcast stream\"}\n\n", clientID)))

	// Gunakan *bufio.Writer
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Keep connection alive
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case message := <-messageChan:
				// Send message to client
				fmt.Fprintf(w, "event: alert\n")
				fmt.Fprintf(w, "data: %s\n\n", message)
				w.Flush() // Flush diperlukan untuk bufio.Writer

			case <-ticker.C:
				// Send heartbeat
				fmt.Fprintf(w, "event: heartbeat\n")
				fmt.Fprintf(w, "data: {\"timestamp\": \"%s\"}\n\n", time.Now().Format(time.RFC3339))
				w.Flush() // Flush diperlukan untuk bufio.Writer

			case <-c.Context().Done():
				return
			}
		}
	})

	return nil
}

// GetSystemLogs returns system activity logs
func GetSystemLogs(c *fiber.Ctx) error {
	var logs []models.SystemLog

	query := database.DB.Preload("User")

	// Filter by user
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	// Pagination
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsedLimit, err := strconv.Atoi(l); err == nil {
			limit = parsedLimit
		}
	}

	offset := 0
	if o := c.Query("offset"); o != "" {
		if parsedOffset, err := strconv.Atoi(o); err == nil {
			offset = parsedOffset
		}
	}

	var total int64
	query.Model(&models.SystemLog{}).Count(&total)

	if err := query.Order("timestamp DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   true,
			"message": "Failed to fetch system logs",
		})
	}

	return c.JSON(fiber.Map{
		"error":  false,
		"data":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Helper function to broadcast to all SSE clients
func broadcastToClients(message string) {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	for _, clientChan := range broadcastClients {
		select {
		case clientChan <- message:
		default:
			// Channel full, skip this client
		}
	}
}

// Helper function to send WhatsApp notifications
func sendWhatsAppNotifications(bencana models.KejadianBencana, message string) {
	// TODO: Integrate with WhatsApp Business API or third-party service
	// This is a placeholder for the actual implementation

	// Get target phone numbers based on bencana level
	var warga []models.WargaRentan
	query := database.DB.Where("no_hp IS NOT NULL AND no_hp != ''")

	if bencana.Level == "Lokal_RT" {
		// Get specific RT/RW numbers
		// query = query.Where("rt = ? AND rw = ?", rt, rw)
	}

	query.Find(&warga)

	// Send notifications (implement actual WA API integration)
	for _, w := range warga {
		// sendWAMessage(w.NoHP, message)
		fmt.Printf("Sending WA to %s: %s\n", w.NoHP, message)
	}
}
