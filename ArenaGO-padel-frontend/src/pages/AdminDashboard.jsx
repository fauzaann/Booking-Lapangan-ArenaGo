import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import Card from "../components/ui/Card";
import Icon from "../components/ui/Icon";
import StatusBadge from "../components/ui/StatusBadge";
import { upcomingDays } from "../lib/mockData";
import { api } from "../lib/api";
import { formatDateInput, formatDateShort, formatIDR } from "../lib/format";

const STATUS_MAP = {
  PENDING: "pending",
  WAITING_PAYMENT: "pending",
  PAID: "confirmed",
  CONFIRMED: "confirmed",
  COMPLETED: "available",
  CANCELLED: "failed",
  EXPIRED: "failed",
};

export default function AdminDashboard() {
  const [dashboard, setDashboard] = useState(null);
  const [bookings, setBookings] = useState([]);
  const [courts, setCourts] = useState([]);
  const [courtId, setCourtId] = useState("");
  const [dayIndex, setDayIndex] = useState(0);
  const [availability, setAvailability] = useState(null);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([api.adminDashboard(), api.adminBookings(), api.adminFields()])
      .then(([dashboardResult, bookingsResult, fieldsResult]) => {
        setDashboard(dashboardResult.data);
        setBookings(bookingsResult.data);
        setCourts(fieldsResult.data);
        setCourtId((current) => current || fieldsResult.data[0]?.id || "");
      })
      .catch((err) => setError(err.message));
  }, []);

  useEffect(() => {
    if (!courtId) return;
    setAvailability(null);
    api.availability(courtId, formatDateInput(upcomingDays[dayIndex]))
      .then((result) => setAvailability(result.data))
      .catch((err) => setError(err.message));
  }, [courtId, dayIndex]);

  const slots = useMemo(() => (availability?.slots || []).map((slot) => ({
    time: slot.start_time,
    status: slot.available ? "available" : "booked",
  })), [availability]);

  const stats = dashboard ? [
    { label: "Total booking", value: dashboard.total_bookings, detail: `${dashboard.confirmed_bookings} terkonfirmasi`, icon: "event_available" },
    { label: "Pendapatan dibayar", value: formatIDR(dashboard.total_revenue), detail: "dari pembayaran PAID", icon: "payments" },
    { label: "Menunggu pembayaran", value: dashboard.pending_bookings, detail: "perlu dipantau", icon: "hourglass_empty" },
    { label: "Lapangan aktif", value: dashboard.total_fields, detail: `${dashboard.total_users} pengguna terdaftar`, icon: "sports_tennis" },
  ] : [];

  if (error) return <div className="px-5 py-20 text-center text-sm text-red-600">{error}</div>;
  if (!dashboard) return <div className="px-5 py-20 text-center text-sm text-ink-muted">Memuat dashboard...</div>;

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <div className="mb-8 flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
        <div>
          <p className="mb-2 text-xs font-bold uppercase tracking-[0.12em] text-champagne">ArenaGO Admin</p>
          <h1 className="font-display text-3xl sm:text-4xl">Ringkasan operasional</h1>
          <p className="mt-2 text-ink-muted">Pantau booking, pendapatan, dan ketersediaan lapangan hari ini.</p>
        </div>
        <Link to="/admin/transaksi" className="inline-flex items-center gap-2 self-start rounded-full bg-onyx px-4 py-2.5 text-sm font-medium text-canvas hover:bg-onyx-hover sm:self-auto">
          <Icon name="receipt_long" size={17} />
          Lihat transaksi
        </Link>
      </div>

      <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {stats.map((stat) => (
          <Card key={stat.label} padded className="p-5">
            <div className="mb-5 flex items-start justify-between">
              <p className="max-w-[12rem] text-xs font-bold uppercase tracking-[0.1em] text-ink-faint">{stat.label}</p>
              <span className="flex h-9 w-9 items-center justify-center rounded-full bg-sage-tint text-sage"><Icon name={stat.icon} size={18} /></span>
            </div>
            <p className="mb-1 font-display text-2xl">{stat.value}</p>
            <p className="text-xs text-ink-muted">{stat.detail}</p>
          </Card>
        ))}
      </div>

      <div className="mb-8 grid grid-cols-1 gap-6 xl:grid-cols-[1.25fr_0.75fr]">
        <Card>
          <div className="mb-5 flex items-center justify-between gap-4">
            <div>
              <h2 className="font-display text-xl">Booking terbaru</h2>
              <p className="mt-1 text-sm text-ink-muted">Aktivitas reservasi terakhir dari semua pelanggan.</p>
            </div>
            <Link to="/admin/transaksi" className="shrink-0 text-sm font-medium text-sage hover:underline">Semua transaksi</Link>
          </div>
          {bookings.length === 0 ? (
            <p className="py-10 text-center text-sm text-ink-faint">Belum ada booking.</p>
          ) : (
            <div className="divide-y divide-border">
              {bookings.map((booking) => (
                <div key={booking.id} className="flex flex-col gap-3 py-4 first:pt-0 sm:flex-row sm:items-center sm:justify-between">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold">{booking.booking_code}</p>
                    <p className="mt-1 truncate text-xs text-ink-muted">{booking.field_name || `Lapangan #${booking.field_id}`} · {booking.booking_date} · {booking.start_time}-{booking.end_time}</p>
                  </div>
                  <div className="flex items-center justify-between gap-4 sm:justify-end">
                    <span className="text-sm font-medium">{formatIDR(booking.total_price)}</span>
                    <StatusBadge status={STATUS_MAP[booking.status] || "confirmed"} label={formatStatus(booking.status)} />
                  </div>
                </div>
              ))}
            </div>
          )}
        </Card>

        <Card>
          <div className="mb-5">
            <h2 className="font-display text-xl">Status booking</h2>
            <p className="mt-1 text-sm text-ink-muted">Distribusi status reservasi saat ini.</p>
          </div>
          <div className="space-y-4">
            <StatusRow label="Menunggu pembayaran" value={dashboard.pending_bookings} total={dashboard.total_bookings} color="bg-champagne" />
            <StatusRow label="Terkonfirmasi" value={dashboard.confirmed_bookings} total={dashboard.total_bookings} color="bg-sage" />
            <StatusRow label="Selesai" value={dashboard.completed_bookings} total={dashboard.total_bookings} color="bg-azure" />
            <StatusRow label="Dibatalkan / expired" value={dashboard.cancelled_bookings + dashboard.expired_bookings} total={dashboard.total_bookings} color="bg-terracotta" />
          </div>
        </Card>
      </div>

      <section>
        <div className="mb-5">
          <p className="mb-2 text-xs font-bold uppercase tracking-[0.12em] text-champagne">Monitoring lapangan</p>
          <h2 className="font-display text-2xl">Ketersediaan slot</h2>
          <p className="mt-1 text-sm text-ink-muted">Pilih lapangan dan tanggal untuk melihat jadwal aktual.</p>
        </div>
        <div className="mb-4 flex flex-wrap gap-2.5">
          {courts.map((court) => (
            <button key={court.id} onClick={() => setCourtId(court.id)} className={`rounded-full border px-4 py-2 text-sm font-medium transition-colors ${courtId === court.id ? "border-onyx bg-onyx text-canvas" : "border-border bg-surface-raised text-ink-muted hover:text-ink"}`}>
              {court.name}
            </button>
          ))}
        </div>
        <div className="mb-5 flex gap-2.5 overflow-x-auto pb-2 no-scrollbar">
          {upcomingDays.map((date, index) => (
            <button key={index} onClick={() => setDayIndex(index)} className={`flex min-w-[76px] flex-col items-center gap-1 rounded-md border px-3 py-3 ${dayIndex === index ? "border-onyx bg-onyx text-canvas" : "border-border bg-surface-raised text-ink hover:border-sage"}`}>
              <span className="text-[11px] uppercase tracking-wide opacity-80">{formatDateShort(date).split(",")[0]}</span>
              <span className="font-display text-lg leading-none">{date.getDate()}</span>
            </button>
          ))}
        </div>
        <Card>
          <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
            <h3 className="font-display text-xl">{availability?.field_name || "Jadwal lapangan"}</h3>
            <div className="flex items-center gap-4 text-xs text-ink-faint"><Legend color="bg-surface-raised border border-border" label="Tersedia" /><Legend color="bg-slate/40" label="Terisi" /></div>
          </div>
          <div className="grid grid-cols-3 gap-2.5 sm:grid-cols-4 lg:grid-cols-7">
            {slots.map((slot) => <div key={slot.time} className={`flex h-16 flex-col items-center justify-center rounded-md border text-sm font-medium ${slot.status === "booked" ? "border-transparent bg-slate/20 text-ink-faint" : "border-border bg-surface-raised text-ink"}`}><span>{slot.time}</span><span className="mt-0.5 text-[10px] font-normal uppercase tracking-wide opacity-70">{slot.status === "booked" ? "Terisi" : "Tersedia"}</span></div>)}
          </div>
        </Card>
      </section>
    </div>
  );
}

function StatusRow({ label, value, total, color }) {
  const percentage = total > 0 ? Math.round((value / total) * 100) : 0;
  return <div><div className="mb-1.5 flex justify-between text-sm"><span className="text-ink-muted">{label}</span><span className="font-medium">{value}</span></div><div className="h-2 overflow-hidden rounded-full bg-surface"><div className={`h-full rounded-full ${color}`} style={{ width: `${percentage}%` }} /></div></div>;
}

function formatStatus(status) {
  return { PENDING: "Pending", WAITING_PAYMENT: "Menunggu", PAID: "Dibayar", CONFIRMED: "Terkonfirmasi", COMPLETED: "Selesai", CANCELLED: "Dibatalkan", EXPIRED: "Expired" }[status] || status;
}

function Legend({ color, label }) {
  return <span className="flex items-center gap-1.5"><span className={`h-3 w-3 rounded-[3px] ${color}`} />{label}</span>;
}
