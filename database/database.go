// database/database.go
package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/HENGKIDWI/Sistem-Mitigasi-Bencana-Berbasis-Komunitas.git/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDB() {
	var err error

	// Database configuration
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	// GORM configuration
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Connection pool settings
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatal("Failed to get database instance:", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("✅ Database connected successfully")
}

func CloseDB() {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Println("Error getting database instance:", err)
		return
	}
	sqlDB.Close()
	log.Println("Database connection closed")
}

func AutoMigrate() {
	err := DB.AutoMigrate(
		&models.User{},
		&models.WargaRentan{},
		&models.KejadianBencana{},
		&models.LogEvakuasi{},
		&models.SystemLog{},
		&models.MasterKecamatan{},
		&models.AdminKota{},
		&models.RekapDataWilayah{},
		&models.MonitoringBencanaKota{},
	)

	if err != nil {
		log.Fatal("Failed to auto migrate:", err)
	}

	log.Println("✅ Auto migration completed")
}
