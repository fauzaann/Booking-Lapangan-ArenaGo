import { Link } from "react-router-dom";
import { useEffect, useState } from "react";
import Icon from "../components/ui/Icon";
import Card from "../components/ui/Card";
import StatusBadge from "../components/ui/StatusBadge";
import Button from "../components/ui/Button";
import { api } from "../lib/api";
import { formatIDR } from "../lib/format";

const STATUS_MAP = {
  confirmed: "confirmed",
  completed: "available",
  cancelled: "failed",
};

const STATUS_LABEL = {
  confirmed: "Terkonfirmasi",
  completed: "Selesai",
  cancelled: "Dibatalkan",
};

export default function MyBookings() {
  const [bookings, setBookings] = useState([]);
  const [error, setError] = useState("");

  useEffect(() => {
    api.bookings({ page: "1", limit: "50" })
      .then((result) => setBookings(result.data.map(mapBooking)))
      .catch((err) => setError(err.message));
  }, []);

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <h1 className="mb-2 font-display text-3xl sm:text-4xl">Riwayat Booking</h1>
      <p className="mb-8 text-ink-muted">Semua reservasi lapangan yang pernah Anda buat.</p>

      {error ? (
        <div className="rounded-lg border border-dashed border-border py-20 text-center text-sm text-red-600">{error}</div>
      ) : bookings.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-20 text-center">
          <Icon name="event_busy" size={32} className="mb-3 text-ink-faint" />
          <p className="mb-1 font-display text-xl">Belum ada booking</p>
          <p className="mb-5 text-sm text-ink-muted">Mulai reservasi lapangan pertama Anda.</p>
          <Link to="/">
            <Button>Eksplor Lapangan</Button>
          </Link>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 xl:grid-cols-3">
          {bookings.map((b) => (
            <Card key={b.id} hover>
              <div className="mb-3 flex items-start justify-between">
                <div>
                  <p className="text-xs uppercase tracking-[0.1em] text-ink-faint">{b.id}</p>
                  <h3 className="font-display text-lg">{b.court.name}</h3>
                </div>
                <StatusBadge status={STATUS_MAP[b.status]} label={STATUS_LABEL[b.status]} />
              </div>
              <div className="mb-4 space-y-1.5 text-sm text-ink-muted">
                <p className="flex items-center gap-1.5">
                  <Icon name="calendar_today" size={14} />
                  {new Date(b.date).toLocaleDateString("id-ID", {
                    weekday: "long",
                    day: "numeric",
                    month: "long",
                  })}
                </p>
                <p className="flex items-center gap-1.5">
                  <Icon name="schedule" size={14} />
                  {b.slots.join(", ")}
                </p>
                <p className="flex items-center gap-1.5">
                  <Icon name="location_on" size={14} />
                  {b.court.location}
                </p>
              </div>
              <div className="flex items-center justify-between border-t border-border pt-3">
                <span className="font-display text-lg">{formatIDR(b.total)}</span>
                {b.status === "confirmed" && (
                  <Link to="/e-tiket" state={{ booking: sampleBookingFrom(b) }}>
                    <Button variant="secondary" size="sm">
                      Lihat E-Tiket
                    </Button>
                  </Link>
                )}
                {b.status === "completed" && (
                  <Link to={`/jadwal/${b.court.id}`}>
                    <Button variant="ghost" size="sm">
                      Booking Lagi
                    </Button>
                  </Link>
                )}
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

function mapBooking(booking) {
  return {
    id: booking.booking_code,
    backendId: booking.id,
    court: {
      id: String(booking.field_id),
      name: booking.field_name || `Lapangan #${booking.field_id}`,
      location: "",
      pricePerHour: booking.items?.[0]?.price || booking.total_price / Math.max(booking.duration, 1),
    },
    date: booking.booking_date,
    slots: [booking.start_time, booking.end_time],
    status: booking.status.toLowerCase(),
    total: booking.total_price,
  };
}

function sampleBookingFrom(b) {
  return {
    id: b.id,
    court: b.court,
    date: new Date(b.date),
    slots: b.slots,
    rental: { rackets: 0, balls: 0 },
    customer: { name: "Raka Pratama", phone: "", email: "" },
  };
}
