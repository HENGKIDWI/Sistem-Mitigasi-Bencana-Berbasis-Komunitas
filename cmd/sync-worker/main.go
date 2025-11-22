// cmd/sync-worker/main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/database"
	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/models"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- Struct untuk Konfigurasi ---

type KecamatanAPIConfig struct {
	KecamatanID uint   `json:"kecamatan_id"`
	Nama        string `json:"nama"`
	ApiURL      string `json:"api_url"`
}

type SyncConfig struct {
	SyncIntervalMinutes int                  `json:"sync_interval_minutes"`
	KecamatanAPIs       []KecamatanAPIConfig `json:"kecamatan_apis"`
}

// --- Struct untuk Respons API Kecamatan ---

type SummaryResponseData struct {
	TotalWarga   int `json:"total_warga"`
	TotalRentan  int `json:"total_rentan"`
	BencanaAktif int `json:"bencana_aktif"`
}

type SummaryResponse struct {
	Data SummaryResponseData `json:"data"`
}

// --- Fungsi Utama ---

func main() {
	// 1. Load .env (untuk koneksi DB Kota)
	if err := godotenv.Load(); err != nil {
		log.Println("Info: No .env file found, relying on environment variables")
	}

	// 2. Load sync_config.json
	config, err := loadConfig("sync_config.json") // Pastikan path sesuai
	if err != nil {
		// Coba path relatif jika dijalankan dari root
		config, err = loadConfig("cmd/sync-worker/sync_config.json")
		if err != nil {
			log.Fatalf("FATAL: Gagal memuat sync_config.json: %v", err)
		}
	}

	// 3. Hubungkan ke Database KOTA
	database.ConnectDB()
	db := database.DB

	// 4. Jalankan sinkronisasi pertama kali
	log.Println("🚀 Worker dimulai. Menjalankan sinkronisasi awal...")
	runSyncJob(db, config)

	// 5. Mulai ticker
	interval := time.Duration(config.SyncIntervalMinutes) * time.Minute
	ticker := time.NewTicker(interval)
	log.Printf("⏱️  Jadwal sync diatur setiap %d menit.", config.SyncIntervalMinutes)

	// 6. Loop utama
	for range ticker.C {
		log.Println("\n⏰ Waktunya sinkronisasi terjadwal...")
		runSyncJob(db, config)
	}
}

func loadConfig(path string) (*SyncConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config SyncConfig
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

// runSyncJob adalah fungsi inti yang melakukan ETL
func runSyncJob(db *gorm.DB, config *SyncConfig) {
	berhasil := 0
	gagal := 0

	for _, kec := range config.KecamatanAPIs {
		// LOG 1: Memberitahu kita sedang menghubungi siapa
		log.Printf("🔄 Menghubungi: %-30s [%s]", kec.Nama, kec.ApiURL)

		// 1. Panggil API /summary
		summaryURL := fmt.Sprintf("%s/summary", kec.ApiURL)

		// Set timeout agar worker tidak 'hang' jika server kecamatan lambat
		client := http.Client{
			Timeout: 5 * time.Second,
		}

		resp, err := client.Get(summaryURL)
		if err != nil {
			// LOG ERROR
			log.Printf("   ❌ GAGAL: Tidak bisa terhubung (%v)", err)
			gagal++
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("   ⚠️  WARNING: Merespon dengan status %d", resp.StatusCode)
			gagal++
			continue
		}

		// 2. Decode JSON
		var summary SummaryResponse
		if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
			log.Printf("   ❌ ERROR: Gagal decode JSON (%v)", err)
			gagal++
			continue
		}

		// 3. Simpan ke DB Kota
		syncRekapWilayah(db, kec.KecamatanID, &summary.Data)
		syncMonitoringBencana(db, kec.KecamatanID, &summary.Data)

		// LOG SUKSES (INI YANG ANDA CARI)
		log.Printf("   ✅ SUKSES: Data tersimpan (Warga: %d, Bencana: %d)",
			summary.Data.TotalWarga, summary.Data.BencanaAktif)
		berhasil++
	}

	log.Printf("📊 Laporan Sync: %d Berhasil, %d Gagal", berhasil, gagal)
	log.Println("---------------------------------------------------------------")
}

func syncRekapWilayah(db *gorm.DB, kecID uint, data *SummaryResponseData) {
	now := time.Now()
	rekap := models.RekapDataWilayah{
		KecamatanID:     kecID,
		TotalWarga:      data.TotalWarga,
		TotalKerentanan: data.TotalRentan,
		LastSync:        &now,
	}

	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "kecamatan_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_warga", "total_kerentanan", "last_sync"}),
	}).Create(&rekap)
}

func syncMonitoringBencana(db *gorm.DB, kecID uint, data *SummaryResponseData) {
	statusLevel := "Aman" // Default aman jika 0 bencana
	if data.BencanaAktif > 5 {
		statusLevel = "Awas"
	} else if data.BencanaAktif > 2 {
		statusLevel = "Siaga"
	} else if data.BencanaAktif > 0 {
		statusLevel = "Waspada"
	}

	monitor := models.MonitoringBencanaKota{
		KecamatanID:  kecID,
		JenisBencana: "Agregat",
		StatusLevel:  statusLevel,
		WaktuLaporan: time.Now(),
		TotalBencana: data.BencanaAktif,
	}

	db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "kecamatan_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"status_level", "waktu_laporan", "total_bencana"}),
	}).Create(&monitor)
}
