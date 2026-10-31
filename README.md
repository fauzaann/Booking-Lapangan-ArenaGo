# Booking-Lapangan-ArenaGo

## Akun Admin

Jalankan migrasi dan seeder terlebih dahulu:

```bash
go run . -migrate-only
go run . -seed
```

Akun admin bawaan:

- Email: `admin@example.com`
- Password: `admin123`

Setelah login, buka `/admin` untuk melihat dashboard operasional. Role admin dari API (`ADMIN`) sudah ditangani secara case-insensitive oleh frontend.

## Menjalankan Lokal

Jalankan backend dari root repository:

```bash
go run main.go
```

Di terminal lain, jalankan frontend dari folder frontend:

```bash
cd ArenaGO-padel-frontend
npm run dev
```

Frontend development memakai proxy `/api` ke backend `http://127.0.0.1:8080`, sehingga halaman Riwayat Booking dan Kelola Jadwal tidak perlu mengakses API lintas-origin secara langsung.

# Booking Field API

Backend REST API **Sistem Booking Lapangan Olahraga** dengan integrasi payment gateway **Xendit**. Dibangun sebagai project akhir Bootcamp Golang dengan penekanan pada correctness, security, dan maintainability — bukan sekadar CRUD yang jalan.

---

## Daftar Isi

1. [Deskripsi](#deskripsi)
2. [Fitur](#fitur)
3. [Tech Stack](#tech-stack)
4. [Arsitektur](#arsitektur)
5. [Struktur Folder](#struktur-folder)
6. [Skema Database](#skema-database)
7. [Setup Environment](#setup-environment)
8. [Instalasi & Menjalankan](#instalasi--menjalankan)
9. [Menjalankan dengan Docker](#menjalankan-dengan-docker)
10. [Dokumentasi API](#dokumentasi-api)
11. [Authentication](#authentication)
12. [Alur Booking](#alur-booking)
13. [Alur Pembayaran & Webhook Xendit](#alur-pembayaran--webhook-xendit)
14. [Contoh Penggunaan (cURL)](#contoh-penggunaan-curl)
15. [Testing](#testing)
16. [Catatan Keputusan Desain](#catatan-keputusan-desain)

---

## Deskripsi

Aplikasi ini melayani pemesanan lapangan padel secara online: user memilih lapangan, mengecek ketersediaan slot per jam, membuat booking, lalu membayar melalui invoice Xendit. Status booking hanya berubah menjadi `CONFIRMED` setelah webhook Xendit tervalidasi di sisi server.

## Fitur

**User**

- Register & login (JWT)
- Melihat daftar lapangan dengan filter (`type`, `status`, `location`, `min_price`, `max_price`, `search`) dan pagination
- Melihat detail lapangan, jadwal operasional, dan ketersediaan slot per tanggal
- Cache Redis opsional untuk daftar dan detail lapangan (TTL default 5 menit); PostgreSQL tetap menjadi sumber data utama
- Membuat booking dengan pengecekan overlap waktu
- Menyimpan beberapa booking/lapangan dalam keranjang dan membayar semuanya dengan satu invoice; slot baru divalidasi dan dikunci saat checkout
- Mendapatkan `payment_url` (invoice Xendit) dan memantau status pembayaran
- Melihat riwayat & detail booking miliknya sendiri
- Mendapat e-tiket dengan QR unik setelah pembayaran terkonfirmasi
- Membatalkan booking sesuai aturan
- Memberi satu rating 1–5 dan ulasan opsional untuk setiap booking berstatus `COMPLETED`
- Melihat rata-rata rating dan ulasan publik lapangan; identitas penulis disamarkan
- Kotak masuk notifikasi in-app dan pengingat jadwal bermain (H-24 & H-2 jam sebelum main) berbasis MongoDB

**Admin**

- Dashboard statistik (user, lapangan, booking per status, revenue)
- CRUD lapangan padel + pengaturan harga, status, fasilitas, dan `image_url`
- Mengatur jam operasional per hari
- Melihat seluruh booking, mengubah status booking
- Memindai QR e-tiket pengguna untuk mencatat check-in satu kali
- Melihat seluruh pembayaran dan data user
- Memoderasi ulasan dengan menyembunyikan atau menampilkan kembali ulasan
- Monitoring log pengingat jadwal bermain (`reminder_jobs` MongoDB)

**Sistem**

### Gambar lapangan

Gambar disimpan sebagai URL, bukan binary di database. Saat membuat atau
mengubah lapangan melalui endpoint admin, isi `image_url` dengan URL publik
dari object storage/CDN, misalnya:

```json
{
  "name": "Lapangan Padel A",
  "type": "PADEL",
  "location": "Jakarta Selatan",
  "price_per_hour": 200000,
  "image_url": "https://cdn.example.com/arenago/padel-a.webp"
}
```

Frontend akan menampilkan gambar tersebut pada kartu lapangan. Jika kosong,
UI otomatis memakai gradient fallback.

- Availability check anti-overlap + advisory lock (anti double booking)
- Database transaction untuk booking + payment
- Webhook idempotent dengan verifikasi ulang ke Xendit
- QR check-in memakai token acak, hanya diterbitkan untuk booking terbayar dan dikonfirmasi
- Worker background: booking kedaluwarsa → `EXPIRED`, booking selesai → `COMPLETED`
- Response JSON konsisten, centralized error handling, request logging

## Tech Stack

| Komponen | Pilihan |
| --- | --- |
| Bahasa | Go 1.25 |
| HTTP Framework | Gin |
| Database Utama | PostgreSQL 16 (GORM) |
| Database Notifikasi & Antrean | MongoDB (mongo-driver/v2) |
| Cache Lapangan | Redis (go-redis/v9, opsional) |
| Auth | JWT (golang-jwt/v5) + bcrypt |
| Validasi | go-playground/validator/v10 |
| Config | godotenv (`.env`) |
| Payment Gateway | Xendit Invoice API |
| Chatbot Assistant | OpenRouter API |
| Dokumentasi | OpenAPI 3.0 + Swagger UI |
| Container | Docker + Docker Compose |

## Arsitektur

```
Request
   ↓
Router  ──► Middleware (CORS → Logger → Recovery → JWT → Role)
   ↓
Handler        (parsing request, validasi payload, menulis response)
   ↓
Controller     (business logic; tidak tahu HTTP maupun GORM)
   ├──► Redis cache (daftar/detail lapangan; cache-aside)
   └──► Repository (interface + implementasi GORM) ──► PostgreSQL
```

PostgreSQL tetap menjadi sumber kebenaran. Cache hanya mencakup daftar/detail lapangan; availability, booking, dan pembayaran tidak di-cache. Perubahan lapangan atau jadwal menginvalidasi cache. Jika Redis tidak tersedia, API tetap membaca dari PostgreSQL dan mencatat error cache.

Untuk pembayaran:

```
Handler → Controller → Xendit Service (pkg/xendit) → Xendit API
```

Prinsip yang dijaga:

- Handler **tidak pernah** melakukan query database atau memanggil Xendit langsung.
- Controller hanya bergantung pada **interface** (`repository.UnitOfWork`, `xendit.InvoiceService`) sehingga mudah di-mock.
- Seluruh dependency dirakit di `main.go` (dependency injection), bukan lewat global variable.

## Struktur Folder

```
booking-field/
├── main.go                     # entrypoint + dependency injection + graceful shutdown
├── config/                     # env, koneksi database, client Xendit
├── database/                   # migration & seeder
├── models/                     # entity GORM
├── dto/                        # request/response API (model DB tidak pernah dikirim langsung)
├── repository/                 # interface + implementasi GORM, UnitOfWork (transaksi)
├── controller/                 # business logic + unit test
├── handler/                    # lapisan HTTP
├── middleware/                 # auth, role, logger, CORS, recovery
├── router/                     # pemetaan endpoint
├── worker/                     # background job (expirer)
├── pkg/
│   ├── apperror/               # error domain + HTTP status
│   ├── jwt/                    # generate & verify token
│   ├── password/               # bcrypt
│   ├── response/               # format response standar
│   ├── timeutil/               # parsing & perhitungan jam booking
│   ├── validator/              # validator + rule kustom (clock, dateonly, notpast)
│   └── xendit/                 # client Invoice API
├── docs/                       # openapi.yaml + Swagger UI
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── .env.example
```

## Skema Database

| Tabel | Kolom Utama |
| --- | --- |
| `users` | id, name, email (unique), password (bcrypt), phone, role, timestamps |
| `fields` | id, name, description, type, location, price_per_hour, facilities, status, deleted_at |
| `schedules` | id, field_id, day, open_time, close_time, is_closed — unique (field_id, day) |
| `bookings` | id, booking_code (unique), user_id, field_id, booking_date, start_time, end_time, duration, total_price, status, notes, cancelled_at, check_in_token (unique), checked_in_at, checked_in_by |
| `ratings` | id, booking_id (unique), user_id, field_id, rating (1–5), review (maks. 1.000 karakter), is_visible |
| `booking_items` | id, booking_id, start_time, end_time, price (rincian per jam) |
| `payments` | id, booking_id (unique), external_id (unique), invoice_id, invoice_url, amount, payment_method, payment_channel, status, paid_at, expired_at |

Relasi: `users 1—N bookings`, `fields 1—N bookings`, `fields 1—N schedules`, `bookings 1—N booking_items`, `bookings 1—1 payments`.

Jam disimpan sebagai string `"HH:MM"` 24 jam agar perbandingan leksikografis di SQL identik dengan perbandingan kronologis.

**Status booking:** `PENDING` → `WAITING_PAYMENT` → `PAID`/`CONFIRMED` → `COMPLETED`, dengan cabang `CANCELLED` dan `EXPIRED`.

## Setup Environment

```bash
cp .env.example .env
```

Isi minimal yang wajib diubah:

```env
JWT_SECRET=<minimal 16 karakter, acak>
XENDIT_SECRET_KEY=<secret key dari dashboard Xendit>
XENDIT_WEBHOOK_TOKEN=<callback verification token dari dashboard Xendit>
```

`.env` sudah masuk `.gitignore` — jangan pernah di-commit. Aplikasi menolak start bila `JWT_SECRET` kurang dari 16 karakter, dan menolak start di `APP_ENV=production` bila kredensial Xendit kosong.

Redis untuk cache lapangan bersifat opsional. Default koneksi adalah `localhost:6379`; jika Redis belum tersedia, backend tetap berjalan dengan fallback PostgreSQL. Atur `REDIS_ADDR`, `REDIS_PASSWORD`, dan `REDIS_DB` untuk koneksi yang berbeda. `FIELD_CACHE_TTL_MINUTES` mengatur TTL (default 5 menit). Untuk mengaktifkan cache lokal, jalankan Redis di alamat tersebut sebelum backend; periksa log `redis: connected` saat startup.

## Instalasi & Menjalankan

```bash
git clone <repo-url>
cd booking-field

cp .env.example .env

# unduh dependency & buat go.sum
go mod tidy

# siapkan database (sekali saja)
createdb booking_field

# jalankan
go run .
```

Server berjalan di `http://localhost:8080`. Migrasi berjalan otomatis (`APP_AUTO_MIGRATE=true`), seeder berjalan bila `RUN_SEEDER=true`.

Perintah lain:

```bash
go run . -migrate-only   # hanya migrasi
go run . -seed           # hanya seeder
make test                # unit test
make cover               # test + laporan coverage
make build               # binary ke bin/booking-field
```

**Akun hasil seeder**

| Role | Email | Password |
| --- | --- | --- |
| ADMIN | admin@example.com | admin123 |
| USER | user@example.com | user123 |

Seeder juga membuat 5 lapangan (Futsal, Badminton, Basket, Tennis, Mini Soccer) beserta jadwal Senin–Minggu. Password di-hash dengan bcrypt, tidak pernah disimpan plaintext.

## Menjalankan dengan Docker

```bash
cp .env.example .env
docker compose up --build
```

Compose menjalankan dua service: `postgres` (dengan healthcheck) dan `app`. Aplikasi baru start setelah database siap.

```bash
docker compose down -v   # hentikan dan hapus volume
```

## Dokumentasi API

Swagger UI: **http://localhost:8080/docs**
Spesifikasi mentah: `http://localhost:8080/docs/openapi.yaml` (file: `docs/openapi.yaml`).

### Ringkasan Endpoint

| Method | Endpoint | Akses |
| --- | --- | --- |
| POST | `/api/v1/auth/register` | Publik |
| POST | `/api/v1/auth/login` | Publik |
| GET | `/api/v1/me` | User |
| GET | `/api/v1/fields` | Publik |
| GET | `/api/v1/fields/:id` | Publik |
| GET | `/api/v1/fields/:id/availability?date=` | Publik |
| GET | `/api/v1/fields/:id/schedules` | Publik |
| GET | `/api/v1/fields/:id/ratings/summary` | Publik |
| GET | `/api/v1/fields/:id/ratings?page=1&limit=10` | Publik |
| POST | `/api/v1/bookings` | User |
| GET | `/api/v1/bookings` | User |
| GET | `/api/v1/bookings/:id` | User |
| GET | `/api/v1/bookings/:id/payment` | User |
| GET/POST | `/api/v1/bookings/:id/review` | Pemilik booking |
| DELETE | `/api/v1/bookings/:id` | User |
| POST | `/api/v1/checkouts` | User |
| GET | `/api/v1/checkouts/:id` | User |
| POST | `/api/v1/checkouts/:id/sync-payment` | User |
| GET | `/api/v1/notifications` | User |
| GET | `/api/v1/notifications/unread-count` | User |
| PATCH | `/api/v1/notifications/:id/read` | User |
| PATCH | `/api/v1/notifications/read-all` | User |
| POST | `/api/v1/payments/webhook` | Xendit (x-callback-token) |
| GET | `/api/v1/admin/dashboard` | Admin |
| GET | `/api/v1/admin/users` | Admin |
| GET/POST | `/api/v1/admin/fields` | Admin |
| PUT/DELETE | `/api/v1/admin/fields/:id` | Admin |
| PUT | `/api/v1/admin/fields/:id/schedules` | Admin |
| GET | `/api/v1/admin/bookings` | Admin |
| PATCH | `/api/v1/admin/bookings/:id/status` | Admin |
| POST | `/api/v1/admin/check-in` | Admin |
| GET | `/api/v1/admin/payments` | Admin |
| GET | `/api/v1/admin/ratings?is_visible=true` | Admin |
| PATCH | `/api/v1/admin/ratings/:id/moderation` | Admin |
| GET | `/api/v1/admin/reminders` | Admin |

### Format Response

Sukses:

```json
{ "success": true, "message": "Success", "data": {} }
```

Sukses dengan pagination:

```json
{
  "success": true,
  "message": "Fields retrieved",
  "data": [],
  "meta": { "page": 1, "limit": 10, "total_items": 5, "total_pages": 1 }
}
```

Error:

```json
{ "success": false, "message": "selected time slot is not available" }
```

Error validasi (422):

```json
{
  "success": false,
  "message": "validation failed",
  "error": { "end_time": "end_time must use HH:MM 24-hour format" }
}
```

Status code yang dipakai: `200`, `201`, `400`, `401`, `403`, `404`, `409`, `422`, `500`, `502`.

Check-in menerima `booking_code` dan `check_in_token` dari QR e-tiket. Endpoint hanya menerima admin, memverifikasi pembayaran dan status booking di dalam transaksi, lalu menolak pemindaian ulang tiket yang sudah digunakan.

## Authentication

Login mengembalikan JWT berisi claims `user_id`, `email`, `role`, `iat`, `exp`. Kirim di setiap request yang membutuhkan autentikasi:

```
Authorization: Bearer <token>
```

Otorisasi role dijaga middleware `RequireRole(models.RoleAdmin)` pada seluruh grup `/api/v1/admin` — user biasa mendapat `403`.

Rating hanya dapat dibuat oleh pemilik booking yang sudah `COMPLETED`; satu booking hanya dapat dinilai sekali. `user_id` dan `field_id` selalu diambil dari data booking di server. Ulasan publik yang disembunyikan admin tidak dihitung dalam ringkasan rating. Pengguna tidak dapat mengedit atau menghapus rating pada MVP.

Daftar lapangan publik menyertakan `average_rating` dan `total_reviews` agar rating dapat langsung ditampilkan pada kartu lapangan di halaman beranda. Lapangan yang belum memiliki ulasan mengembalikan nilai `0` untuk keduanya.

## Alur Booking

```
Pilih lapangan → pilih tanggal → pilih jam
        ↓
Server: validasi payload (format, tanggal tidak lampau, end > start)
        ↓
Server: cek lapangan aktif + jam berada dalam jadwal operasional
        ↓
BEGIN TRANSACTION
   advisory lock (field_id + tanggal)
   cek overlap slot          → bentrok? 409 Conflict
   hitung durasi & harga     (price_per_hour × durasi)
   generate booking_code     BK-YYYYMMDD-XXXX
   insert booking + booking_items
   insert payment (PENDING)
COMMIT
        ↓
Create Xendit Invoice        → gagal? booking otomatis CANCELLED, response 502
        ↓
BEGIN TRANSACTION
   simpan invoice_url & expired_at
   booking → WAITING_PAYMENT
COMMIT
        ↓
Response 201 + payment_url
```

### Keranjang dan checkout gabungan

Keranjang frontend hanya menyimpan draft lokal dan tidak mengunci slot. Saat
`POST /api/v1/checkouts` dipanggil, backend mengunci pasangan lapangan/tanggal
secara konsisten, mengecek ulang jam operasional dan semua overlap, lalu membuat
seluruh booking dan satu invoice Xendit dalam satu checkout.

Contoh request:

```json
{
  "items": [
    {
      "item_id": "cart-item-1",
      "field_id": 1,
      "booking_date": "2030-01-15",
      "start_time": "10:00",
      "end_time": "12:00"
    },
    {
      "item_id": "cart-item-2",
      "field_id": 2,
      "booking_date": "2030-01-15",
      "start_time": "13:00",
      "end_time": "14:00"
    }
  ]
}
```

Harga dihitung dari data lapangan di server. Jika satu atau lebih slot sudah
terisi saat checkout, seluruh checkout dibatalkan dan response `409` menyertakan
`error.conflicts` dengan `item_id` agar item yang bentrok dapat ditandai di
keranjang. Tidak ada invoice atau booking parsial yang dibuat. Pembayaran satu
invoice memperbarui semua payment dan booking dalam checkout secara atomik.
Status checkout dapat dibaca kembali dan disinkronkan melalui endpoint di atas.

**Aturan overlap** menggunakan interval half-open `[start, end)`:
`existing.start < request.end AND existing.end > request.start`.
Sehingga existing `10:00–12:00` vs request `11:00–13:00` **ditolak**, sedangkan `10:00–11:00` dan `11:00–12:00` **boleh berdampingan**.

**Pembatalan:** booking `PENDING`/`WAITING_PAYMENT` bebas dibatalkan. Booking `PAID`/`CONFIRMED` hanya bisa dibatalkan user minimal `BOOKING_CANCEL_MIN_HOUR` (default 24) jam sebelum jam mulai; admin tidak dibatasi.

Pengguna yang login dapat membatalkan booking melalui `POST /api/v1/bookings/:id/cancel` (opsional `{"reason":"..."}`); `DELETE /api/v1/bookings/:id` tetap tersedia. Booking `CANCELLED`, `COMPLETED`, `EXPIRED`, atau yang sudah dimulai tidak dapat dibatalkan melalui alur pengguna. Pembayaran `PAID` tidak otomatis direfund dan respons booking menandai `refund_review_required` untuk peninjauan manual.

Perubahan jadwal tersedia melalui `PATCH /api/v1/bookings/:id/reschedule` dengan `booking_date`, `start_time`, dan `end_time`. Perubahan harus mengikuti jam operasional dan durasi yang berlaku, slot tujuan harus tersedia, dan tanggal/jam mulai tidak boleh sudah lewat. Untuk booking yang sudah dibayar, batas waktu `BOOKING_CANCEL_MIN_HOUR` juga berlaku. Jika tarif baru berbeda dari tagihan tersimpan, perubahan ditolak karena penyesuaian invoice/refund belum tersedia. Histori jadwal disimpan bersama booking; pengingat dan notifikasi diperbarui setelah transaksi PostgreSQL berhasil.

## Alur Pembayaran & Webhook Xendit

```
Xendit
   ↓ POST /api/v1/payments/webhook  (header x-callback-token)
Validasi token (constant time compare)     → salah? 401
   ↓
Verifikasi ulang invoice ke Xendit API     (server-side verification)
   ↓
Cari payment berdasarkan external_id       → tidak ada? 404
   ↓
Sudah final (PAID/EXPIRED/FAILED)?         → ya: abaikan (idempotent), 200
   ↓
Cocokkan nominal yang dibayar              → kurang? 400
   ↓
payment = PAID, paid_at, method, channel
booking = CONFIRMED
```

Status pembayaran **tidak pernah** diambil dari frontend. Endpoint webhook bersifat idempotent: callback berulang untuk invoice yang sama tidak menghasilkan perubahan ganda.

### Mengatur webhook di dashboard Xendit

1. Buka **Settings → Developers → Webhooks**.
2. Isi URL invoice paid/expired: `https://<domain-anda>/api/v1/payments/webhook`.
3. Salin **Callback Verification Token** ke `XENDIT_WEBHOOK_TOKEN`.
4. Untuk development lokal, ekspos server dengan tunneling (misal ngrok) lalu daftarkan URL publiknya.

## Contoh Penggunaan (cURL)

**1. Register**

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com","password":"user123","phone":"081234567890"}'
```

**2. Login**

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"user123"}'
```

```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2026-08-30T10:00:00Z",
    "user": { "id": 2, "name": "John Doe", "email": "user@example.com", "phone": "081200000002", "role": "USER" }
  }
}
```

**3. Cek ketersediaan**

```bash
curl "http://localhost:8080/api/v1/fields/1/availability?date=2026-09-01"
```

```json
{
  "success": true,
  "message": "Availability retrieved",
  "data": {
    "field_id": 1,
    "field_name": "Lapangan Futsal Arena A",
    "date": "2026-09-01",
    "day": "TUESDAY",
    "is_open": true,
    "open_time": "08:00",
    "close_time": "23:00",
    "price_per_hour": 150000,
    "slots": [
      { "start_time": "10:00", "end_time": "11:00", "price": 150000, "available": false, "reason": "already booked" },
      { "start_time": "11:00", "end_time": "12:00", "price": 150000, "available": true }
    ]
  }
}
```

**4. Membuat booking**

```bash
curl -X POST http://localhost:8080/api/v1/bookings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"field_id":1,"booking_date":"2026-09-01","start_time":"10:00","end_time":"12:00"}'
```

```json
{
  "success": true,
  "message": "Booking created",
  "data": {
    "booking_id": 1,
    "booking_code": "BK-20260829-0001",
    "field_name": "Lapangan Futsal Arena A",
    "booking_date": "2026-09-01",
    "start_time": "10:00",
    "end_time": "12:00",
    "duration": 2,
    "total_price": 300000,
    "booking_status": "WAITING_PAYMENT",
    "payment_url": "https://checkout.xendit.co/web/...",
    "payment_status": "PENDING",
    "expired_at": "2026-08-29T11:00:00Z"
  }
}
```

Slot bentrok akan menghasilkan `409`:

```json
{ "success": false, "message": "selected time slot is not available" }
```

**5. Simulasi webhook (development)**

```bash
curl -X POST http://localhost:8080/api/v1/payments/webhook \
  -H "Content-Type: application/json" \
  -H "x-callback-token: $XENDIT_WEBHOOK_TOKEN" \
  -d '{"external_id":"BK-20260829-0001","status":"PAID","amount":300000,"paid_amount":300000,"payment_method":"EWALLET","payment_channel":"OVO"}'
```

**6. Dashboard admin**

```bash
curl http://localhost:8080/api/v1/admin/dashboard -H "Authorization: Bearer $ADMIN_TOKEN"
```

## Testing

```bash
go test ./...
go test ./... -v
go test ./... -cover
```

Unit test berjalan tanpa database maupun koneksi internet — seluruh repository dan Xendit di-mock lewat interface (`controller/mock_test.go`).

Cakupan pengujian:

| Area | Kasus |
| --- | --- |
| Authentication | register berhasil, password tersimpan sebagai bcrypt, email duplikat (409), login berhasil, password salah (401), email tidak dikenal tetap 401 |
| Booking | booking berhasil (harga & kode benar), lapangan tidak ditemukan (404), lapangan non-aktif (422), slot sudah dibooking (409), tanggal lampau (422), rentang jam tidak valid (422), di luar jam operasional (422), melebihi durasi maksimum (422), booking orang lain tidak terbaca (404), pembatalan terlalu dekat jam mulai (422), pembatalan booking belum dibayar |
| Payment | webhook token salah (401), webhook PAID → booking CONFIRMED, webhook duplikat (idempotent), nominal kurang (400), webhook EXPIRED, external_id tidak dikenal (404), verifikasi status ke Xendit |
| Time utility | deteksi overlap 7 skenario, perhitungan durasi, batas jam operasional, pemecahan slot per jam, format jam |

## Catatan Keputusan Desain

**Pemanggilan Xendit di luar transaksi database.** Spesifikasi meminta booking dan payment dibuat dalam satu transaksi. Yang dilakukan: booking + booking_items + payment (`PENDING`) disimpan atomik dalam satu transaksi, lalu invoice dibuat, lalu transaksi kedua menyimpan `invoice_url`. Menahan transaksi database selama HTTP request eksternal berlangsung akan mengunci koneksi dan baris terlalu lama. Konsistensi tetap terjaga karena kegagalan invoice otomatis membatalkan booking.

**Advisory lock, bukan sekadar `SELECT ... FOR UPDATE`.** Baris yang perlu dikunci belum ada saat booking baru dibuat, sehingga row lock tidak menolong. `pg_advisory_xact_lock` pada kombinasi `field_id + tanggal` membuat dua request bersamaan untuk slot yang sama diproses berurutan, dan lock otomatis lepas saat commit/rollback.

**Client Xendit ditulis manual.** `pkg/xendit` memakai `net/http` standar (Basic Auth dengan secret key sebagai username) agar dependency tetap sedikit dan perilaku timeout/error handling terlihat jelas. Menggantinya dengan SDK resmi cukup mengganti implementasi `xendit.InvoiceService` tanpa menyentuh controller.

**Swagger tanpa code generation.** Dokumentasi ditulis sebagai `docs/openapi.yaml` dan di-embed ke binary (`go:embed`), sehingga tidak perlu menjalankan `swag init` setiap kali build dan dokumentasi tetap tersedia di dalam container.

**Jam sebagai string `HH:MM`.** Perbandingan string zero-padded 24 jam setara dengan perbandingan waktu, sehingga query overlap tetap sederhana dan dapat memakai index biasa.

---

## Checklist Fitur

- [x] Struktur project modular (handler, controller, repository, model, dto, middleware, config, router)
- [x] Konfigurasi `.env` + validasi konfigurasi saat start
- [x] Models & migration + index tambahan
- [x] Repository pattern berbasis interface + UnitOfWork untuk transaksi
- [x] Register/Login, JWT, bcrypt
- [x] Middleware auth, role, logger, CORS, recovery
- [x] CRUD lapangan, filter, pagination, pengaturan jadwal
- [x] Availability check anti-overlap per jam
- [x] Booking system + booking code unik + perhitungan harga di backend
- [x] Integrasi Xendit Invoice (tanpa fake payment)
- [x] Webhook Xendit: validasi token, idempotent, verifikasi server-side
- [x] Admin dashboard, manajemen booking, pembayaran, dan user
- [x] Swagger/OpenAPI
- [x] Dockerfile + Docker Compose
- [x] Unit test (auth, booking, payment, time utility)
- [x] README

## Smoke test

Jalankan PostgreSQL, backend, dan frontend dev server terlebih dahulu. Dengan
`XENDIT_WEBHOOK_TOKEN` yang sama seperti backend, jalankan:

```bash
cd ArenaGO-padel-frontend
npm run smoke:e2e
```

Smoke test memeriksa halaman frontend, health API, register/login, field,
dashboard admin, membuat booking, menolak token webhook yang salah, lalu
memastikan webhook PAID mengubah payment menjadi `PAID` dan booking menjadi
`CONFIRMED`. Gunakan `SMOKE_PAYMENT=false` untuk hanya menjalankan smoke test
tanpa membuat booking/payment.

### Troubleshooting pembayaran

Pembayaran nyata membutuhkan API key Xendit yang aktif. Isi `.env` dengan:

```env
XENDIT_SECRET_KEY=xnd_production_atau_xnd_development_key
XENDIT_WEBHOOK_TOKEN=token_callback_dari_xendit
XENDIT_BASE_URL=https://api.xendit.co
```

Pastikan key valid sebelum menjalankan backend. Pemeriksaan aman yang tidak
membuat invoice baru:

```bash
set -a && . ./.env && set +a
curl -u "$XENDIT_SECRET_KEY:" -o /dev/null -sS -w "%{http_code}\n" \
  "$XENDIT_BASE_URL/v2/invoices/nonexistent-invoice"
```

Status `401` atau `403` berarti API key ditolak dan harus diganti dari dashboard
Xendit. Status `404` berarti autentikasi berhasil dan invoice contoh memang tidak
ditemukan.

Untuk E2E lokal tanpa kredensial Xendit sungguhan, jalankan mock gateway:

```bash
node scripts/mock-xendit.mjs
```

Jalankan backend dengan `XENDIT_SECRET_KEY=mock-secret`,
`XENDIT_WEBHOOK_TOKEN=smoke-webhook-token`, dan
`XENDIT_BASE_URL=http://127.0.0.1:9090` sebelum menjalankan smoke test.

Untuk menguji pembuatan invoice Xendit development tanpa memalsukan status
paid:

```bash
FRONTEND_URL=http://127.0.0.1:5173 \
SMOKE_PAYMENT=true SMOKE_WEBHOOK=false \
npm --prefix ArenaGO-padel-frontend run smoke:e2e
```

Hasil yang benar pada tahap ini adalah booking `WAITING_PAYMENT`, payment
`PENDING`, dan URL checkout Xendit. Status `CONFIRMED`/`PAID` hanya boleh
terjadi setelah user menyelesaikan checkout dan Xendit mengirim webhook asli.

INFO LEBIH LANJUT : https://drive.google.com/drive/folders/11H8wMfZIOl6ceDfmBWmSkYpCEQL6165U?usp=sharing
