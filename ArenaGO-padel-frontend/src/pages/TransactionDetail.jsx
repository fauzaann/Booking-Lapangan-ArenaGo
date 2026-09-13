import { useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import Icon from "../components/ui/Icon";
import Card from "../components/ui/Card";
import StatusBadge from "../components/ui/StatusBadge";
import { api } from "../lib/api";
import { formatIDR } from "../lib/format";

export default function TransactionDetail() {
  const { id } = useParams();
  const [booking, setBooking] = useState(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api.adminBooking(id)
      .then((result) => setBooking(result.data))
      .catch((err) => setError(err.message));
  }, [id]);

  if (error) return <div className="px-5 py-20 text-center text-sm text-red-600">{error}</div>;
  if (!booking) return <div className="px-5 py-20 text-center text-sm text-ink-muted">Memuat detail transaksi...</div>;

  const payment = booking.payment;
  const paymentStatus = payment?.status || "PENDING";

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <Link
        to="/admin/transaksi"
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink"
      >
        <Icon name="arrow_back" size={18} />
        Kembali ke Riwayat
      </Link>

      <div className="mb-8 flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="mb-1 text-xs uppercase tracking-[0.12em] text-ink-faint">
            {booking.booking_code}
          </p>
          <h1 className="font-display text-3xl">{formatIDR(payment?.amount || booking.total_price)}</h1>
        </div>
        <StatusBadge status={mapPaymentStatus(paymentStatus)} label={formatPaymentStatus(paymentStatus)} className="mt-1" />
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_1fr]">
        <div className="space-y-6">
          <Card>
            <h2 className="mb-4 font-display text-xl">Detail Booking</h2>
            <dl className="space-y-3 text-sm">
              <Row label="Booking ID" value={booking.booking_code} />
              <Row label="Pelanggan" value={booking.user?.name || `User #${booking.user_id}`} />
              <Row label="Lapangan" value={booking.field_name || `Lapangan #${booking.field_id}`} />
              <Row label="Jadwal" value={`${booking.booking_date} · ${booking.start_time}-${booking.end_time}`} />
              <Row label="Metode Pembayaran" value={payment?.payment_method || payment?.payment_channel || "Belum dibayar"} />
              <Row label="Dibuat" value={new Date(booking.created_at).toLocaleString("id-ID")} />
              <Row
                label="Dibayar"
                value={payment?.paid_at ? new Date(payment.paid_at).toLocaleString("id-ID") : "—"}
              />
            </dl>
          </Card>

          <Card>
            <h2 className="mb-4 font-display text-xl">Data Pembayaran</h2>
            <pre className="overflow-x-auto rounded-md bg-onyx px-4 py-4 text-xs leading-relaxed text-canvas/90">
{JSON.stringify(payment || {}, null, 2)}
            </pre>
          </Card>
        </div>

        <Card>
          <h2 className="mb-5 font-display text-xl">Log Webhook</h2>
          <div className="flex items-center gap-4 text-sm text-ink-muted">
            <span className="flex h-7 w-7 items-center justify-center rounded-full bg-sage-tint text-sage"><Icon name="database" size={14} /></span>
            <p>Status payment tersimpan di database.</p>
          </div>
          {payment?.invoice_id ? <p className="mt-5 break-all text-xs text-ink-faint">Invoice ID: {payment.invoice_id}</p> : null}
        </Card>
      </div>
    </div>
  );
}

function mapPaymentStatus(status) {
  if (status === "PAID") return "success";
  if (status === "PENDING") return "pending";
  return "failed";
}

function formatPaymentStatus(status) {
  return { PAID: "Berhasil", PENDING: "Menunggu Pembayaran", FAILED: "Gagal", EXPIRED: "Expired" }[status] || status;
}

function Row({ label, value }) {
  return (
    <div className="flex justify-between gap-4 border-b border-border pb-3 last:border-b-0 last:pb-0">
      <dt className="text-ink-faint">{label}</dt>
      <dd className="text-right font-medium">{value}</dd>
    </div>
  );
}
