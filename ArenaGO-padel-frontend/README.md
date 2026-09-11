# ArenaGO — Frontend UI

Frontend React (Vite) untuk sistem booking lapangan padel "ArenaGO".
Data lapangan, autentikasi, booking, pembayaran, dan concierge menggunakan backend API.

## Menjalankan

```bash
npm install
npm run dev
```

Buka `http://localhost:3000`.

## Struktur

```
src/
  context/
    AuthContext.jsx  sesi JWT dari backend RBAC
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
    Login               /login       masuk melalui backend
    Register            /register    daftar akun melalui backend
    Confirmation        /konfirmasi  data pemesan + sewa tambahan (perlu login)
    Payment             /pembayaran  redirect ke invoice Xendit (perlu login)
    ETicket             /e-tiket     e-tiket setelah pembayaran (perlu login)
    MyBookings          /riwayat     riwayat booking milik user (perlu login)
    Profile             /profil      data akun + logout (perlu login)
    TransactionList     /admin/transaksi       riwayat transaksi semua user (admin)
    TransactionDetail   /admin/transaksi/:id   detail + log webhook (admin)
    AdminDashboard      /admin       kelola jadwal/slot (admin)
  lib/
    mockData.js  fallback/demo data dan konfigurasi UI
    api.js       client backend API dan adapter response
    format.js    helper format IDR & tanggal
```

## Auth

`AuthContext` menyimpan JWT dan user session di `localStorage`.

`RequireAuth` menangani redirect ke `/login` dan validasi role admin.

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
- Payment flow melalui invoice Xendit
- Floating concierge terhubung ke endpoint OpenRouter backend

