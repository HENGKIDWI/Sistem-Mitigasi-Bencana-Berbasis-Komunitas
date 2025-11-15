# ============================================
# Panduan Lengkap
# ============================================

# Sistem Mitigasi Bencana - Arsitektur Terdistribusi

## 🏗️ Arsitektur

Sistem ini menggunakan **arsitektur terdistribusi** dengan 3 komponen utama:

### 1. API Kecamatan (Per Wilayah)
- Berjalan di setiap kecamatan
- Database lokal per kecamatan
- Handle: Warga, Bencana, Evakuasi lokal
- Port default: 3001, 3002, 3003, dst.

### 2. API Kota (Pusat)
- Berjalan di server Pemkot
- Database agregasi kota
- Handle: Monitoring city-wide, Reports
- Port default: 4000

### 3. Sync Worker (ETL Service)
- Background service tanpa API
- Sinkronisasi data dari kecamatan → kota
- Interval: 5 menit (configurable)

## 📁 Struktur Project

```
mitigasi-bencana-backend/
├── cmd/
│   ├── api-kecamatan/
│   │   ├── main.go
│   │   └── .env
│   ├── api-kota/
│   │   ├── main.go
│   │   └── .env
│   └── sync-worker/
│       ├── main.go
│       └── sync_config.json
├── database/
│   └── database.go
├── models/
│   └── models.go
├── middleware/
│   └── auth.go
├── handlers/
│   ├── handler_kecamatan.go
│   ├── handler_kota.go
│   ├── warga.go
│   ├── bencana.go
│   └── monitoring.go
└── go.mod
```

## 🚀 Installation & Setup

### 1. Setup Databases

```sql
-- Buat database untuk setiap kecamatan
CREATE DATABASE mitigasi_bencana_kec_bangkalan;
CREATE DATABASE mitigasi_bencana_kec_socah;
CREATE DATABASE mitigasi_bencana_kec_burneh;

-- Buat database kota (pusat)
CREATE DATABASE mitigasi_bencana_kota;
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Setup Environment

```bash
# API Kecamatan 1 (Bangkalan)
cd cmd/api-kecamatan
cp ../../.env.kecamatan_example .env
# Edit .env sesuai konfigurasi

# API Kota
cd ../api-kota
cp ../../.env.kota_example .env
# Edit .env sesuai konfigurasi

# Sync Worker
cd ../sync-worker
cp ../../sync_config.json.example sync_config.json
# Edit sync_config.json
```

### 4. Run Services

```bash
# Terminal 1 - API Kecamatan Bangkalan
cd cmd/api-kecamatan
go run main.go

# Terminal 2 - API Kota
cd cmd/api-kota
go run main.go

# Terminal 3 - Sync Worker
cd cmd/sync-worker
go run main.go
```

## 📡 API Endpoints

### API Kecamatan (Port 3001)

#### Authentication
- `POST /api/v1/auth/login` - Login RT/RW/Relawan
- `POST /api/v1/auth/register` - Register user baru
- `GET /api/v1/auth/me` - Get profile

#### Warga Rentan
- `GET /api/v1/warga` - List warga (filter: rt, rw, kategori)
- `POST /api/v1/warga` - Tambah warga (RT/RW)
- `PUT /api/v1/warga/:id` - Update warga
- `DELETE /api/v1/warga/:id` - Hapus warga

#### Bencana
- `GET /api/v1/bencana` - List bencana
- `GET /api/v1/bencana/active` - Bencana aktif
- `POST /api/v1/bencana` - Lapor bencana
- `PUT /api/v1/bencana/:id/status` - Update status

#### Evakuasi
- `GET /api/v1/evakuasi/prioritas/:bencana_id` - Daftar prioritas
- `POST /api/v1/evakuasi/log` - Catat evakuasi
- `PUT /api/v1/evakuasi/log/:id` - Update status evakuasi

### API Kota (Port 4000)

#### Authentication
- `POST /api/v1/auth/login` - Login Pemkot/BPBD

#### Master Data
- `GET /api/v1/kecamatan` - List kecamatan
- `POST /api/v1/kecamatan` - Tambah kecamatan

#### Monitoring
- `GET /api/v1/monitoring/kota` - Dashboard kota
- `GET /api/v1/monitoring/kecamatan/:id` - Detail kecamatan
- `GET /api/v1/monitoring/statistik` - Statistik agregat

#### Reports
- `GET /api/v1/reports/dashboard` - Dashboard data
- `GET /api/v1/rekap` - Rekap semua wilayah

## 🔄 Sync Worker

Worker berjalan setiap 5 menit (default) dan melakukan:

1. **Sync Rekap Data Wilayah**
   - Hitung total warga per kecamatan
   - Hitung total kerentanan
   - Update timestamp sync

2. **Sync Monitoring Bencana**
   - Ambil bencana aktif per kecamatan
   - Tentukan status level (Waspada/Siaga/Awas)
   - Update monitoring table di kota

## 🔐 Security

- JWT-based authentication
- Role-based access control
- Separate tokens untuk Kecamatan vs Kota
- Password hashing dengan bcrypt

## 🎯 Best Practices

1. **Isolasi Database**: Setiap kecamatan punya database sendiri
2. **No Direct Access**: Kota tidak akses langsung ke DB kecamatan
3. **ETL Pattern**: Sync worker handle semua transfer data
4. **Graceful Sync**: Jika satu kecamatan down, yang lain tetap jalan
5. **Idempotent**: Sync bisa dijalankan berkali-kali tanpa duplikasi

## 📊 Monitoring

Setiap service memiliki endpoint `/health`:
- API Kecamatan: `GET http://localhost:3001/health`
- API Kota: `GET http://localhost:4000/health`

## 🛠️ Development

```bash
# Run dengan hot reload (install air)
air

# Build
cd cmd/api-kecamatan && go build -o ../../bin/api-kecamatan
cd cmd/api-kota && go build -o ../../bin/api-kota
cd cmd/sync-worker && go build -o ../../bin/sync-worker

# Test
go test ./...
```

## 📝 TODO

- [ ] Implement WhatsApp notification
- [ ] Add retry mechanism pada sync worker
- [ ] Implement circuit breaker
- [ ] Add metrics & logging (Prometheus)
- [ ] Implement backup & disaster recovery
- [ ] Add integration tests
- [ ] Deploy dengan Docker/Kubernetes
