// models/models.go
package models

import (
	"time"

	"gorm.io/gorm"
)

// User model
type User struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	Username     string         `gorm:"unique;not null" json:"username"`
	Password     string         `gorm:"not null" json:"-"`
	Role         string         `gorm:"type:enum('RT','RW','Relawan','Admin_Kecamatan');not null" json:"role"`
	NamaLengkap  string         `gorm:"not null" json:"nama_lengkap"`
	NoHP         string         `json:"no_hp"`
	WilayahTugas string         `json:"wilayah_tugas"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// WargaRentan model
type WargaRentan struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	NIK            string         `gorm:"unique;not null" json:"nik"`
	Nama           string         `gorm:"not null" json:"nama"`
	Alamat         string         `gorm:"type:text" json:"alamat"`
	RT             string         `json:"rt"`
	RW             string         `json:"rw"`
	KategoriRentan string         `gorm:"type:enum('Lansia','Disabilitas','Anak-anak','Ibu Hamil','Sakit Keras','Non-Rentan');not null" json:"kategori_rentan"`
	SkorPrioritas  int            `json:"skor_prioritas"`
	Latitude       float64        `gorm:"type:decimal(10,8)" json:"latitude"`
	Longitude      float64        `gorm:"type:decimal(11,8)" json:"longitude"`
	NoHP           string         `json:"no_hp"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// KejadianBencana model
type KejadianBencana struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	JenisBencana  string         `gorm:"not null" json:"jenis_bencana"`
	Level         string         `gorm:"type:enum('Lokal_RT','Kecamatan');not null" json:"level"`
	WaktuMulai    time.Time      `gorm:"not null" json:"waktu_mulai"`
	WaktuSelesai  *time.Time     `json:"waktu_selesai"`
	Status        string         `gorm:"type:enum('Aktif','Selesai');not null;default:'Aktif'" json:"status"`
	UserPelaporID uint           `gorm:"not null" json:"user_pelapor_id"`
	UserPelapor   User           `gorm:"foreignKey:UserPelaporID" json:"user_pelapor,omitempty"`
	Deskripsi     string         `gorm:"type:text" json:"deskripsi"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// LogEvakuasi model
type LogEvakuasi struct {
	ID            uint            `gorm:"primarykey" json:"id"`
	BencanaID     uint            `gorm:"not null" json:"bencana_id"`
	Bencana       KejadianBencana `gorm:"foreignKey:BencanaID" json:"bencana,omitempty"`
	WargaID       uint            `gorm:"not null" json:"warga_id"`
	Warga         WargaRentan     `gorm:"foreignKey:WargaID" json:"warga,omitempty"`
	RelawanID     uint            `gorm:"not null" json:"relawan_id"`
	Relawan       User            `gorm:"foreignKey:RelawanID" json:"relawan,omitempty"`
	StatusTerkini string          `gorm:"type:enum('Menunggu','Dalam Proses','Teevakuasi','Di Titik Kumpul');not null" json:"status_terkini"`
	WaktuUpdate   time.Time       `gorm:"not null" json:"waktu_update"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

// SystemLog model
type SystemLog struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"not null" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Aktivitas string    `gorm:"not null" json:"aktivitas"`
	Timestamp time.Time `gorm:"not null;autoCreateTime" json:"timestamp"`
}

// MasterKecamatan model
type MasterKecamatan struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Nama      string         `gorm:"not null" json:"nama"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// AdminKota model
type AdminKota struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Username  string         `gorm:"unique;not null" json:"username"`
	Role      string         `gorm:"type:enum('Pemkot','BPBD');not null" json:"role"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// RekapDataWilayah model
type RekapDataWilayah struct {
	ID              uint            `gorm:"primarykey" json:"id"`
	KecamatanID     uint            `gorm:"not null" json:"kecamatan_id"`
	Kecamatan       MasterKecamatan `gorm:"foreignKey:KecamatanID" json:"kecamatan,omitempty"`
	TotalWarga      int             `gorm:"default:0" json:"total_warga"`
	TotalKerentanan int             `gorm:"default:0" json:"total_kerentanan"`
	LastSync        *time.Time      `json:"last_sync"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// MonitoringBencanaKota model
type MonitoringBencanaKota struct {
	ID           uint            `gorm:"primarykey" json:"id"`
	KecamatanID  uint            `gorm:"not null" json:"kecamatan_id"`
	Kecamatan    MasterKecamatan `gorm:"foreignKey:KecamatanID" json:"kecamatan,omitempty"`
	JenisBencana string          `gorm:"not null" json:"jenis_bencana"`
	StatusLevel  string          `gorm:"type:enum('Waspada','Siaga','Awas');not null" json:"status_level"`
	WaktuLaporan time.Time       `gorm:"not null" json:"waktu_laporan"`
	TotalBencana int             `gorm:"default:0" json:"total_bencana"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// DTO for Login Request
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// DTO for Login Response
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// DTO for Create Warga
type CreateWargaRequest struct {
	NIK            string  `json:"nik" validate:"required"`
	Nama           string  `json:"nama" validate:"required"`
	Alamat         string  `json:"alamat"`
	RT             string  `json:"rt"`
	RW             string  `json:"rw"`
	KategoriRentan string  `json:"kategori_rentan" validate:"required"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	NoHP           string  `json:"no_hp"`
}

// DTO for Create Bencana
type CreateBencanaRequest struct {
	JenisBencana string `json:"jenis_bencana" validate:"required"`
	Level        string `json:"level" validate:"required"`
	Deskripsi    string `json:"deskripsi"`
}

type RegisterRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	Role         string `json:"role"`
	NamaLengkap  string `json:"nama_lengkap"`
	NoHP         string `json:"no_hp"`
	WilayahTugas string `json:"wilayah_tugas"`
}

// DTO for Broadcast Notification
type BroadcastRequest struct {
	BencanaID uint   `json:"bencana_id" validate:"required"`
	Message   string `json:"message" validate:"required"`
	Level     string `json:"level" validate:"required"`
}
