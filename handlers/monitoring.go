// handlers/monitoring.go
package handlers

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/database"
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/models"
	"github.com/gofiber/fiber/v2"
)

// SSE Broadcast management
var (
	broadcastClients = make(map[string]chan string)
	clientsMutex     sync.RWMutex
)

// GetMonitoringKecamatan returns monitoring data for a kecamatan
func GetMonitoringKecamatan(c *fiber.Ctx) error {
	kecamatanID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid kecamatan ID",
		})
	}

	// Get active bencana in kecamatan
	var bencana []models.MonitoringBencanaKota
	database.DB.Where("kecamatan_id = ?", kecamatanID).
		Preload("Kecamatan").
		Order("waktu_laporan DESC").
		Find(&bencana)

	// Get rekap data wilayah
	var rekap models.RekapDataWilayah
	database.DB.Where("kecamatan_id = ?", kecamatanID).
		Preload("Kecamatan").
		First(&rekap)

	// Count active evacuations
	var activeEvacuations int64
	database.DB.Model(&models.LogEvakuasi{}).
		Joins("JOIN kejadian_bencana ON kejadian_bencana.id = log_evakuasi.bencana_id").
		Where("kejadian_bencana.status = ?", "Aktif").
		Where("log_evakuasi.status_terkini NOT IN ?", []string{"Di Titik Kumpul"}).
		Count(&activeEvacuations)

	return c.JSON(fiber.Map{
		"error": false,
		"data": fiber.Map{
			"bencana":            bencana,
			"rekap":              rekap,
			"active_evacuations": activeEvacuations,
		},
	})
}

// GetMonitoringKota returns monitoring data for entire city
func GetMonitoringKota(c *fiber.Ctx) error {
	// Get all monitoring data grouped by kecamatan
	var monitoring []models.MonitoringBencanaKota
	database.DB.Preload("Kecamatan").
		Order("waktu_laporan DESC").
		Limit(50).
		Find(&monitoring)

	// Get rekap data for all kecamatan
	var rekapData []models.RekapDataWilayah
	database.DB.Preload("Kecamatan").Find(&rekapData)

	// Calculate total statistics
	var totalWarga, totalKerentanan int
	for _, rekap := range rekapData {
		totalWarga += rekap.TotalWarga
		totalKerentanan += rekap.TotalKerentanan
	}

	// Get active bencana count
	var activeBencanaCount int64
	database.DB.Model(&models.KejadianBencana{}).
		Where("status = ?", "Aktif").
		Count(&activeBencanaCount)

	return c.JSON(fiber.Map{
		"error": false,
		"data": fiber.Map{
			"monitoring":           monitoring,
			"rekap_kecamatan":      rekapData,
			"total_warga":          totalWarga,
			"total_kerentanan":     totalKerentanan,
			"active_bencana_count": activeBencanaCount,
		},
	})
}

// GetRekapWilayah returns rekap data for a specific kecamatan
func GetRekapWilayah(c *fiber.Ctx) error {
	kecamatanID, err := strconv.Atoi(c.Params("kecamatan_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   true,
			"message": "Invalid kecamatan ID",
		})
	}

	var rekap models.RekapDataWilayah
	if err := database.DB.Where("kecamatan_id = ?", kecamatanID).
		Preload("Kecamatan").
		First(&rekap).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   true,
			"message": "Rekap not found",
		})
	}

	// Get detailed breakdown by kategori
	var breakdown []struct {
		KategoriRentan string `json:"kategori_rentan"`
		Total          int    `json:"total"`
	}

	database.DB.Model(&models.WargaRentan{}).
		Select("kategori_rentan, COUNT(*) as total").
		Group("kategori_rentan").
		Scan(&breakdown)

	return c.JSON(fiber.Map{
		"error": false,
		"data": fiber.Map{
			"rekap":     rekap,
			"breakdown": breakdown,
		},
	})
}

// SendDaruratNotification sends emergency notification
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
		"🚨 PERINGATAN DARURAT 🚨\n\nJenis: %s\nLevel: %s\nPesan: %s\nWaktu: %s",
		bencana.JenisBencana,
		bencana.Level,
		req.Message,
		time.Now().Format("02/01/2006 15:04"),
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
	c.Context().SetBodyStreamWriter(func(w *fiber.Writer) {
		// Keep connection alive
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case message := <-messageChan:
				// Send message to client
				fmt.Fprintf(w, "event: alert\n")
				fmt.Fprintf(w, "data: %s\n\n", message)
				w.Flush()

			case <-ticker.C:
				// Send heartbeat
				fmt.Fprintf(w, "event: heartbeat\n")
				fmt.Fprintf(w, "data: {\"timestamp\": \"%s\"}\n\n", time.Now().Format(time.RFC3339))
				w.Flush()

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
