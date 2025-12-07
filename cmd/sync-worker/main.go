package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/database"
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/models"
	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

// Struktur Pesan yang diterima dari Kafka
type EventMessage struct {
	Action      string                 `json:"action"`
	KecamatanID uint                   `json:"kecamatan_id"`
	Payload     map[string]interface{} `json:"payload"`
}

func main() {
	// 1. Setup Database Kota
	if err := godotenv.Load(); err != nil {
		log.Println("Info: No .env file found")
	}
	database.ConnectDB()

	// 2. Konfigurasi Kafka Reader (Consumer)
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"localhost:9092"}, // Konek ke Docker
		Topic:    "sync-events",              // Topic harus sama dengan Producer
		GroupID:  "kota-sync-worker-group",   // PENTING: Group ID agar offset tersimpan
		MinBytes: 10e3,                       // 10KB
		MaxBytes: 10e6,                       // 10MB
	})
	defer reader.Close()

	log.Println("🚀 Sync Worker Berjalan (Event-Driven Mode)... Menunggu Event dari Kafka...")

	// 3. Loop Abadi (Mendengarkan Stream)
	for {
		// ReadMessage akan 'block' (berhenti tunggu) sampai ada pesan masuk
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("❌ Error baca pesan: %v", err)
			break
		}

		// Ada pesan masuk!
		log.Printf("📨 Event Masuk: %s", string(m.Key))
		processEvent(database.DB, m.Value)
	}
}

func processEvent(db *gorm.DB, messageBytes []byte) {
	var event EventMessage
	if err := json.Unmarshal(messageBytes, &event); err != nil {
		log.Printf("❌ Gagal decode JSON: %v", err)
		return
	}

	// Router Logic berdasarkan Action
	switch event.Action {
	case "CREATE_BENCANA":
		log.Printf("⚡ Bencana terdeteksi di Kecamatan ID %d. Mengupdate Monitoring Kota...", event.KecamatanID)
		updateMonitoringKota(db, event.KecamatanID)

	case "CREATE_WARGA":
		log.Printf("bust Warga bertambah di Kecamatan ID %d. Mengupdate Rekap...", event.KecamatanID)
		// Implementasi update rekap di sini

	default:
		log.Printf("⚠️ Action tidak dikenal: %s", event.Action)
	}
}

// Fungsi Update DB Kota (Versi Sederhana: Increment)
func updateMonitoringKota(db *gorm.DB, kecID uint) {
	// Upsert logika untuk menambah jumlah bencana
	// Karena ini event driven, idealnya payload berisi data lengkap
	// Tapi untuk simpel, kita increment saja count-nya

	// Cek apakah data monitoring sudah ada
	var monitor models.MonitoringBencanaKota
	result := db.Where("kecamatan_id = ?", kecID).First(&monitor)

	if result.Error == gorm.ErrRecordNotFound {
		monitor = models.MonitoringBencanaKota{
			KecamatanID:  kecID,
			JenisBencana: "Update Terbaru",
			StatusLevel:  "Waspada",
			TotalBencana: 1,
		}
		db.Create(&monitor)
	} else {
		// Update existing
		monitor.TotalBencana += 1
		monitor.StatusLevel = "Siaga" // Contoh logika
		db.Save(&monitor)
	}
	log.Println("✅ Database Kota Terupdate!")
}
