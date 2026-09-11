// All data below is static mock/dummy data used purely to drive the UI.
// No backend, no real API — this file is the single source of truth for
// every screen so the demo stays consistent end to end.

export const COURT_GRADIENTS = [
  "from-[#2D4B3E] to-[#14171A]",
  "from-[#1E3A5F] to-[#14171A]",
  "from-[#725B38] to-[#14171A]",
  "from-[#3A4B4B] to-[#14171A]",
];

export const courts = [
  {
    id: "crt-01",
    name: "Court Sage — Panoramic",
    location: "Senayan, Jakarta",
    type: "Indoor Panoramic",
    surface: "WPT Blue Turf",
    rating: 4.9,
    reviews: 128,
    pricePerHour: 450000,
    tags: ["AC", "Kaca Panorama", "Sewa Raket"],
    gradient: COURT_GRADIENTS[0],
    vip: true,
  },
  {
    id: "crt-02",
    name: "Court Azure — Tournament",
    location: "Kemang, Jakarta",
    type: "Indoor",
    surface: "Pro Blue Turf",
    rating: 4.8,
    reviews: 96,
    pricePerHour: 400000,
    tags: ["AC", "Live Score", "Tribun"],
    gradient: COURT_GRADIENTS[1],
    vip: false,
  },
  {
    id: "crt-03",
    name: "Court Onyx — Signature",
    location: "Canggu, Bali",
    type: "Outdoor Covered",
    surface: "Sand Green Turf",
    rating: 5.0,
    reviews: 74,
    pricePerHour: 380000,
    tags: ["Bar Lounge", "Sewa Raket"],
    gradient: COURT_GRADIENTS[2],
    vip: true,
  },
  {
    id: "crt-04",
    name: "Court Terrace — Garden",
    location: "Seminyak, Bali",
    type: "Outdoor",
    surface: "Sand Green Turf",
    rating: 4.7,
    reviews: 51,
    pricePerHour: 320000,
    tags: ["Kolam Renang", "Cafe"],
    gradient: COURT_GRADIENTS[3],
    vip: false,
  },
];

export function buildDaySlots() {
  const hours = [
    "07:00", "08:00", "09:00", "10:00", "11:00",
    "13:00", "14:00", "15:00", "16:00", "17:00",
    "18:00", "19:00", "20:00", "21:00",
  ];
  const bookedSet = new Set(["09:00", "10:00", "18:00", "19:00"]);
  const vipSet = new Set(["17:00", "18:00", "19:00", "20:00"]);

  return hours.map((time) => ({
    time,
    end: addOneHour(time),
    status: bookedSet.has(time) ? "booked" : "available",
    vip: vipSet.has(time),
  }));
}

function addOneHour(time) {
  const [h, m] = time.split(":").map(Number);
  const next = (h + 1) % 24;
  return `${String(next).padStart(2, "0")}:${String(m).padStart(2, "0")}`;
}

export const upcomingDays = Array.from({ length: 7 }).map((_, i) => {
  const d = new Date();
  d.setDate(d.getDate() + i);
  return d;
});

export const sampleBooking = {
  id: "APB-240915-0142",
  court: courts[0],
  date: upcomingDays[1],
  slots: ["17:00", "18:00"],
  players: 4,
  rental: { rackets: 2, balls: 1 },
  customer: {
    name: "Raka Pratama",
    phone: "+62 812-3456-7890",
    email: "raka.pratama@email.com",
  },
};

export function bookingPriceBreakdown(booking) {
  const courtTotal = booking.court.pricePerHour * booking.slots.length;
  const racketRental = booking.rental.rackets * 75000;
  const ballRental = booking.rental.balls * 35000;
  const serviceFee = 15000;
  const subtotal = courtTotal + racketRental + ballRental;
  const total = subtotal + serviceFee;
  return [
    { label: `Sewa lapangan (${booking.slots.length} jam)`, amount: courtTotal },
    { label: `Sewa raket (${booking.rental.rackets}x)`, amount: racketRental },
    { label: `Bola padel (${booking.rental.balls} tube)`, amount: ballRental },
    { label: "Biaya layanan", amount: serviceFee },
  ].concat({ label: "__total__", amount: total });
}

export const paymentMethods = [
  {
    group: "QRIS",
    items: [{ id: "qris", name: "QRIS (GoPay, OVO, ShopeePay, Dana)", icon: "qr_code_2" }],
  },
  {
    group: "Virtual Account",
    items: [
      { id: "va-bca", name: "BCA Virtual Account", icon: "account_balance" },
      { id: "va-mandiri", name: "Mandiri Virtual Account", icon: "account_balance" },
      { id: "va-bri", name: "BRI Virtual Account", icon: "account_balance" },
    ],
  },
  {
    group: "Kartu",
    items: [{ id: "card", name: "Kartu Kredit / Debit", icon: "credit_card" }],
  },
];

export const transactions = [
  {
    id: "TXN-98213",
    bookingId: "APB-240915-0142",
    customer: "Raka Pratama",
    court: "Court Sage — Panoramic",
    amount: 1015000,
    method: "QRIS",
    status: "success",
    createdAt: "2026-09-10T09:12:00",
    paidAt: "2026-09-10T09:13:42",
    webhookEvents: [
      { event: "payment.created", status: 200, at: "2026-09-10T09:12:01" },
      { event: "payment.pending", status: 200, at: "2026-09-10T09:12:03" },
      { event: "payment.succeeded", status: 200, at: "2026-09-10T09:13:43" },
    ],
    rawPayload: {
      id: "qr_5f2a1c",
      external_id: "APB-240915-0142",
      status: "PAID",
      amount: 1015000,
      payment_method: "QRIS",
    },
  },
  {
    id: "TXN-98207",
    bookingId: "APB-240914-0098",
    customer: "Dinda Ayu",
    court: "Court Azure — Tournament",
    amount: 815000,
    method: "BCA Virtual Account",
    status: "pending",
    createdAt: "2026-09-10T08:40:00",
    paidAt: null,
    webhookEvents: [
      { event: "payment.created", status: 200, at: "2026-09-10T08:40:02" },
      { event: "payment.pending", status: 200, at: "2026-09-10T08:40:05" },
    ],
    rawPayload: {
      id: "va_88bcaa",
      external_id: "APB-240914-0098",
      status: "PENDING",
      amount: 815000,
      payment_method: "BCA_VA",
    },
  },
  {
    id: "TXN-98180",
    bookingId: "APB-240913-0071",
    customer: "Michael Tanuwijaya",
    court: "Court Onyx — Signature",
    amount: 380000,
    method: "Kartu Kredit",
    status: "failed",
    createdAt: "2026-09-09T19:02:00",
    paidAt: null,
    webhookEvents: [
      { event: "payment.created", status: 200, at: "2026-09-09T19:02:01" },
      { event: "payment.failed", status: 200, at: "2026-09-09T19:04:12" },
    ],
    rawPayload: {
      id: "cc_11f3ba",
      external_id: "APB-240913-0071",
      status: "FAILED",
      amount: 380000,
      payment_method: "CREDIT_CARD",
      failure_code: "INSUFFICIENT_FUNDS",
    },
  },
];

export const adminStats = [
  { label: "Booking hari ini", value: "18", delta: "+4 dari kemarin" },
  { label: "Okupansi lapangan", value: "76%", delta: "+6% minggu ini" },
  { label: "Pendapatan hari ini", value: "IDR 8.140.000", delta: "+12%" },
  { label: "Menunggu pembayaran", value: "3", delta: "perlu ditinjau" },
];

export const conciergeSeedMessages = [
  {
    from: "assistant",
    text: "Selamat datang di Atelier Assistant. Ada yang bisa saya bantu untuk jadwal main padel Anda hari ini?",
  },
];

export const conciergeQuickActions = [
  "Booking Court 3 besok jam 17.00",
  "Sewa raket Bullpadel Pro",
  "Cek jadwal coach minggu ini",
];

// --- Auth & profile mock data -----------------------------------------
// UI-only. The real project already has a backend with RBAC — these
// objects only shape what the frontend expects the API to return, so
// swapping this out for real calls later is a drop-in replacement.

export const mockUsers = {
  customer: {
    id: "usr-1001",
    name: "Raka Pratama",
    email: "raka.pratama@email.com",
    phone: "+62 812-3456-7890",
    role: "customer",
    joinedAt: "2025-11-02",
  },
  admin: {
    id: "usr-9001",
    name: "Sarah Wijaya",
    email: "sarah.wijaya@atelierpadel.com",
    phone: "+62 813-2211-0099",
    role: "admin",
    joinedAt: "2025-01-15",
  },
};

export const myBookings = [
  {
    id: "APB-240915-0142",
    court: courts[0],
    date: "2026-09-15",
    slots: ["17:00", "18:00"],
    status: "confirmed",
    total: 1015000,
  },
  {
    id: "APB-240902-0071",
    court: courts[1],
    date: "2026-09-02",
    slots: ["09:00"],
    status: "completed",
    total: 400000,
  },
  {
    id: "APB-240820-0033",
    court: courts[2],
    date: "2026-08-20",
    slots: ["19:00", "20:00"],
    status: "cancelled",
    total: 760000,
  },
];

