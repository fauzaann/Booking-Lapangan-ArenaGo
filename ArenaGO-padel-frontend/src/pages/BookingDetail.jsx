import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import Card from "../components/ui/Card";
import Button from "../components/ui/Button";
import Icon from "../components/ui/Icon";
import StatusBadge from "../components/ui/StatusBadge";
import { api } from "../lib/api";
import { formatIDR } from "../lib/format";

const BOOKING_STATUS = {
  PENDING: ["pending", "Menunggu pembayaran"],
  WAITING_PAYMENT: ["pending", "Menunggu pembayaran"],
  PAID: ["confirmed", "Dibayar"],
  CONFIRMED: ["confirmed", "Terkonfirmasi"],
  COMPLETED: ["available", "Selesai"],
  CANCELLED: ["failed", "Dibatalkan"],
  EXPIRED: ["failed", "Expired"],
};

export default function BookingDetail() {
  const { id } = useParams();
  const [booking, setBooking] = useState(null);
  const [payment, setPayment] = useState(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    Promise.all([api.booking(id), api.bookingPayment(id)])
      .then(([bookingResult, paymentResult]) => {
        if (!active) return;
        setBooking(bookingResult.data);
        setPayment(paymentResult.data);
      })
      .catch((err) => {
        if (active) setError(err.message);
      });
    return () => { active = false; };
  }, [id]);

  if (error) return <div className="px-5 py-20 text-center text-sm text-red-600">{error}</div>;
  if (!booking) return <div className="px-5 py-20 text-center text-sm text-ink-muted">Memuat detail booking...</div>;

  const [status, statusLabel] = BOOKING_STATUS[booking.status] || ["pending", booking.status];
  const canPay = payment?.status === "PENDING" && payment.invoice_url;

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <Link to="/riwayat" className="mb-6 inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink">
        <Icon name="arrow_back" size={18} />
        Kembali ke Riwayat Booking
      </Link>

      <div className="mb-8 flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="mb-1 text-xs uppercase tracking-[0.12em] text-ink-faint">{booking.booking_code}</p>
          <h1 className="font-display text-3xl">Detail Booking</h1>
          <p className="mt-2 text-sm text-ink-muted">Informasi booking dan status pembayaran terbaru.</p>
        </div>
        <StatusBadge status={status} label={statusLabel} />
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1.1fr_0.9fr]">
        <Card>
          <h2 className="mb-5 font-display text-xl">Informasi Reservasi</h2>
          <dl className="space-y-3 text-sm">
            <Row label="Lapangan" value={booking.field_name || `Lapangan #${booking.field_id}`} />
            <Row label="Tanggal" value={formatDate(booking.booking_date)} />
            <Row label="Jam bermain" value={`${booking.start_time} - ${booking.end_time}`} />
            <Row label="Durasi" value={`${booking.duration} jam`} />
            <Row label="Dibuat" value={formatDateTime(booking.created_at)} />
            <Row label="Catatan" value={booking.notes || "-"} />
          </dl>
        </Card>

        <Card>
          <h2 className="mb-5 font-display text-xl">Pembayaran</h2>
          <dl className="space-y-3 text-sm">
            <Row label="Total" value={formatIDR(booking.total_price)} />
            <Row label="Status" value={formatPaymentStatus(payment?.status)} />
            <Row label="Metode" value={payment?.payment_method || payment?.payment_channel || "Belum dibayar"} />
            <Row label="Dibayar" value={payment?.paid_at ? formatDateTime(payment.paid_at) : "-"} />
            <Row label="Invoice" value={payment?.invoice_id || "-"} />
          </dl>
          {canPay ? (
            <a href={payment.invoice_url} target="_blank" rel="noreferrer" className="mt-6 block">
              <Button className="w-full">Lanjutkan Pembayaran</Button>
            </a>
          ) : null}
          {(booking.status === "CONFIRMED" || booking.status === "PAID") ? (
            <Link to="/e-tiket" state={{ booking: toETicketBooking(booking) }} className="mt-3 block">
              <Button variant="secondary" className="w-full">Lihat E-Tiket</Button>
            </Link>
          ) : null}
        </Card>
      </div>
    </div>
  );
}

function Row({ label, value }) {
  return <div className="flex justify-between gap-4 border-b border-border pb-3 last:border-b-0 last:pb-0"><dt className="text-ink-faint">{label}</dt><dd className="text-right font-medium">{value}</dd></div>;
}

function formatDate(value) {
  return new Intl.DateTimeFormat("id-ID", { weekday: "long", day: "numeric", month: "long", year: "numeric" }).format(new Date(value));
}

function formatDateTime(value) {
  return new Intl.DateTimeFormat("id-ID", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}

function formatPaymentStatus(status) {
  return { PAID: "Berhasil", PENDING: "Menunggu pembayaran", FAILED: "Gagal", EXPIRED: "Expired" }[status] || "Belum tersedia";
}

function toETicketBooking(booking) {
  return {
    id: booking.booking_code,
    court: {
      name: booking.field_name || `Lapangan #${booking.field_id}`,
      location: "-",
      pricePerHour: booking.total_price / Math.max(booking.duration, 1),
    },
    date: new Date(booking.booking_date),
    slots: [booking.start_time, booking.end_time],
    rental: { rackets: 0, balls: 0 },
    totalOverride: booking.total_price,
    customer: { name: "Pemesan", phone: "", email: "" },
  };
}