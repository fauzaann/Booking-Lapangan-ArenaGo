# Atelier Padel — Frontend UI

Frontend-only React (Vite) untuk sistem booking lapangan padel "Atelier Padel".
Seluruh data adalah **mock/dummy data** (`src/lib/mockData.js`) — tidak ada
backend, database, API nyata, autentikasi, atau integrasi pembayaran/webhook
Xendit yang sesungguhnya. Layar pembayaran dan log webhook hanya simulasi UI.

## Menjalankan

```bash
npm install
npm run dev
```

Buka `http://localhost:5173`.

## Struktur

```
src/
  context/
    AuthContext.jsx  sesi mock (login/register/logout), siap diganti ke API RBAC asli
  components/
    ui/         Button, Card, Modal, Navbar, BottomNav, StatusBadge, Tabs, Input, Icon
    booking/    CourtCard, TimeSlotGrid, BookingSummary, ConciergeWidget
    routing/    RequireAuth — guard rute yang butuh login / role tertentu
  layouts/
    AppLayout   Navbar + BottomNav + Concierge + Outlet
    AuthLayout  Layout minimal untuk halaman login/register
  pages/
    Explore             /            eksplor lapangan (publik)
    SelectSlot          /jadwal/:id  pilih tanggal & jam (publik)
    Login               /login       masuk (mock session)
    Register            /register    daftar akun (mock session)
    Confirmation        /konfirmasi  data pemesan + sewa tambahan (perlu login)
    Payment             /pembayaran  UI gateway Xendit (mock, perlu login)
    ETicket             /e-tiket     e-tiket setelah pembayaran (perlu login)
    MyBookings          /riwayat     riwayat booking milik user (perlu login)
    Profile             /profil      data akun + logout (perlu login)
    TransactionList     /admin/transaksi       riwayat transaksi semua user (admin)
    TransactionDetail   /admin/transaksi/:id   detail + log webhook (admin)
    AdminDashboard      /admin       kelola jadwal/slot (admin)
  lib/
    mockData.js  seluruh dummy data (courts, slots, bookings, transactions, users, dsb)
    format.js    helper format IDR & tanggal
```

## Auth (UI-only)

`AuthContext` menyimpan sesi mock di `localStorage` (key `atelier-padel:session`)
supaya login tidak hilang saat refresh — bukan token asli, cuma objek user
dummy. Di halaman Login ada dua tombol "Demo cepat" (Pelanggan / Admin) untuk
gampang cek tampilan role customer vs admin tanpa backend.

**Untuk menyambungkan ke backend RBAC kamu:** ganti isi fungsi `login`,
`register`, dan `updateProfile` di `src/context/AuthContext.jsx` dengan
pemanggilan API asli (fetch/axios ke endpoint kamu). Bentuk objek `user`
(`id, name, email, phone, role`) sudah generik supaya gampang dipetakan ke
response API sungguhan. `RequireAuth` (di `src/components/routing/`) sudah
menangani redirect ke `/login` kalau belum login, dan ke `/` kalau role tidak
sesuai (`role="admin"`).

## Design System

Diambil langsung dari `DESIGN.md` pada file desain asli (Google Stitch export):
warna (onyx/champagne/sage/azure), tipografi Playfair Display + Plus Jakarta
Sans, radius pill untuk komponen interaktif, radius 16-24px untuk kontainer,
dan efek glassmorphism pada navbar & concierge. Token warna/radius/font
didefinisikan sebagai Tailwind v4 `@theme` di `src/index.css`.

## State UI yang sudah disimulasikan

- Loading skeleton & empty state (Eksplor, Riwayat Transaksi)
- Selected / booked / VIP time slot
- Modal konfirmasi blokir slot (Admin)
- Tab & filter status transaksi
- Date & time-slot selection
- Payment flow: idle -> processing -> success (simulasi, tanpa transaksi nyata)
- Floating AI concierge (UI chat, tanpa respons pintar - sesuai scope)

## Yang sengaja TIDAK dibuat

Backend, database, API nyata, autentikasi, integrasi Xendit/payment gateway
nyata, dan webhook nyata - sesuai batasan project (frontend/UI only).
