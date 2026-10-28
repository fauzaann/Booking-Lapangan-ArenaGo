import { useCallback, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import Card from "../components/ui/Card";
import Button from "../components/ui/Button";
import Icon from "../components/ui/Icon";
import StatusBadge from "../components/ui/StatusBadge";
import Modal from "../components/ui/Modal";
import { api } from "../lib/api";
import { formatIDR } from "../lib/format";
import { useAuth } from "../context/AuthContext";

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
  const { user } = useAuth();
  const [booking, setBooking] = useState(null);
  const [payment, setPayment] = useState(null);
  const [rating, setRating] = useState(null);
  const [ratingValue, setRatingValue] = useState(5);
  const [reviewText, setReviewText] = useState("");
  const [ratingSubmitting, setRatingSubmitting] = useState(false);
  const [ratingError, setRatingError] = useState("");
  const [ratingFeedback, setRatingFeedback] = useState("");
  const [error, setError] = useState("");
  const [syncing, setSyncing] = useState(false);
  const [syncFeedback, setSyncFeedback] = useState("");
  const [cancelOpen, setCancelOpen] = useState(false);
  const [cancelReason, setCancelReason] = useState("");
  const [cancelling, setCancelling] = useState(false);
  const [cancelError, setCancelError] = useState("");
  const [cancelFeedback, setCancelFeedback] = useState("");

  const loadData = useCallback(async () => {
    try {
      const [bookingResult, paymentResult, ratingResult] = await Promise.all([
        api.booking(id),
        api.bookingPayment(id),
        api.bookingRating(id).catch((err) => {
          if (err.status === 404) return null;
          throw err;
        }),
      ]);
      setBooking(bookingResult.data);
      setPayment(paymentResult.data);
      setRating(ratingResult?.data || null);
      return { booking: bookingResult.data, payment: paymentResult.data, rating: ratingResult?.data || null };
    } catch (err) {
      setError(err.message);
      return null;
    }
  }, [id]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSyncPayment = async () => {
    setSyncing(true);
    setSyncFeedback("");
    try {
      await api.syncPayment(id);
      const updated = await loadData();
      if (updated?.payment?.status === "PAID" || updated?.booking?.status === "CONFIRMED") {
        setSyncFeedback("Status berhasil disinkronkan: Pembayaran Lunas!");
      } else {
        setSyncFeedback("Status pembayaran masih menunggu. Silakan selesaikan di Xendit.");
      }
    } catch (err) {
      setSyncFeedback(err.message || "Gagal menyinkronkan status.");
    } finally {
      setSyncing(false);
    }
  };

  const handleCancelBooking = async () => {
    setCancelling(true);
    setCancelError("");
    try {
      const result = await api.cancelBooking(id, cancelReason.trim());
      setBooking(result.data);
      setPayment(result.data.payment || payment);
      setCancelOpen(false);
      setCancelReason("");
      setCancelFeedback("Booking berhasil dibatalkan.");
    } catch (err) {
      setCancelError(err.message || "Gagal membatalkan booking.");
    } finally {
      setCancelling(false);
    }
  };

  const handleSubmitRating = async (event) => {
    event.preventDefault();
    setRatingSubmitting(true);
    setRatingError("");
    setRatingFeedback("");
    try {
      const result = await api.createRating(id, { rating: ratingValue, review: reviewText.trim() });
      setRating(result.data);
      setRatingFeedback("Terima kasih, rating dan ulasan Anda berhasil dikirim.");
    } catch (err) {
      setRatingError(err.message || "Gagal mengirim ulasan.");
    } finally {
      setRatingSubmitting(false);
    }
  };

  if (error) return <div className="px-5 py-20 text-center text-sm text-red-600">{error}</div>;
  if (!booking) return <div className="px-5 py-20 text-center text-sm text-ink-muted">Memuat detail booking...</div>;

  const [status, statusLabel] = BOOKING_STATUS[booking.status] || ["pending", booking.status];
  const isPending = booking.status === "PENDING" || booking.status === "WAITING_PAYMENT";
  const canPay = payment?.status === "PENDING" && payment.invoice_url;
  const canCancel = ["PENDING", "WAITING_PAYMENT", "PAID", "CONFIRMED"].includes(booking.status)
    && isBookingInFuture(booking);

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

      {cancelFeedback && (
        <p role="status" className="mb-5 rounded-md bg-emerald-50 px-4 py-3 text-sm text-emerald-800">
          {cancelFeedback}
        </p>
      )}

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
          {syncFeedback && (
            <p className="mt-4 rounded-md bg-canvas px-3 py-2 text-center text-xs text-ink-muted">
              {syncFeedback}
            </p>
          )}

          {canPay ? (
            <a href={payment.invoice_url} target="_blank" rel="noreferrer" className="mt-6 block">
              <Button className="w-full">Lanjutkan Pembayaran</Button>
            </a>
          ) : null}

          {isPending ? (
            <div className="mt-3">
              <Button
                variant="secondary"
                className="w-full"
                onClick={handleSyncPayment}
                disabled={syncing}
              >
                {syncing ? "Memeriksa Status..." : "Cek Status Pembayaran"}
              </Button>
            </div>
          ) : null}

          {(booking.status === "CONFIRMED" || booking.status === "PAID") ? (
            <Link to="/e-tiket" state={{ booking: toETicketBooking(booking, user) }} className="mt-4 block">
              <Button variant="secondary" className="w-full">Lihat E-Tiket</Button>
            </Link>
          ) : null}

          {canCancel && (
            <Button
              variant="danger"
              className="mt-4 w-full"
              onClick={() => {
                setCancelError("");
                setCancelOpen(true);
              }}
            >
              Batalkan Booking
            </Button>
          )}
        </Card>
      </div>

      {booking.status === "COMPLETED" && (
        <Card className="mt-6">
          <h2 className="mb-2 font-display text-xl">Rating & ulasan lapangan</h2>
          <p className="mb-5 text-sm text-ink-muted">
            Bagikan pengalaman Anda untuk membantu pemain lain memilih lapangan.
          </p>
          {rating ? (
            <div className="rounded-md bg-canvas p-4">
              <div className="flex items-center gap-1" aria-label={`Rating ${rating.rating} dari 5`}>
                {Array.from({ length: 5 }, (_, index) => (
                  <Icon key={index} name="star" size={19} filled={index < rating.rating} className={index < rating.rating ? "text-champagne" : "text-border"} />
                ))}
                <span className="ml-2 text-sm font-medium">{rating.rating}/5</span>
              </div>
              {rating.review && <p className="mt-3 whitespace-pre-line text-sm text-ink-muted">{rating.review}</p>}
              <p className="mt-3 text-xs text-ink-faint">Ulasan Anda sudah dikirim dan tidak dapat diubah.</p>
            </div>
          ) : (
            <form onSubmit={handleSubmitRating}>
              <fieldset>
                <legend className="mb-2 text-sm font-medium">Pilih rating</legend>
                <div className="flex gap-1" role="radiogroup" aria-label="Rating lapangan">
                  {Array.from({ length: 5 }, (_, index) => {
                    const value = index + 1;
                    return (
                      <button
                        key={value}
                        type="button"
                        role="radio"
                        aria-checked={ratingValue === value}
                        aria-label={`${value} bintang`}
                        onClick={() => setRatingValue(value)}
                        className="rounded p-1 focus-visible:outline focus-visible:outline-2 focus-visible:outline-onyx"
                      >
                        <Icon name="star" size={28} filled={value <= ratingValue} className={value <= ratingValue ? "text-champagne" : "text-border"} />
                      </button>
                    );
                  })}
                </div>
              </fieldset>
              <label htmlFor="review-text" className="mb-2 mt-5 block text-sm font-medium">
                Ulasan (opsional)
              </label>
              <textarea
                id="review-text"
                value={reviewText}
                onChange={(event) => setReviewText(event.target.value)}
                maxLength={1000}
                rows={4}
                placeholder="Ceritakan pengalaman bermain Anda..."
                className="w-full rounded-md border border-border bg-surface px-4 py-3 text-sm focus:border-onyx focus:outline-none"
              />
              <p className="mt-1 text-right text-xs text-ink-faint">{reviewText.length}/1000</p>
              {ratingError && <p role="alert" className="mt-3 text-sm text-red-600">{ratingError}</p>}
              {ratingFeedback && <p role="status" className="mt-3 text-sm text-emerald-700">{ratingFeedback}</p>}
              <Button type="submit" className="mt-4" disabled={ratingSubmitting}>
                {ratingSubmitting ? "Mengirim..." : "Kirim ulasan"}
              </Button>
            </form>
          )}
        </Card>
      )}

      <Modal
        open={cancelOpen}
        onClose={() => {
          if (!cancelling) setCancelOpen(false);
        }}
        title="Batalkan Booking"
        footer={
          <div className="flex justify-end gap-3">
            <Button variant="secondary" size="sm" onClick={() => setCancelOpen(false)} disabled={cancelling}>
              Kembali
            </Button>
            <Button variant="danger" size="sm" onClick={handleCancelBooking} disabled={cancelling}>
              {cancelling ? "Membatalkan..." : "Ya, Batalkan"}
            </Button>
          </div>
        }
      >
        <p className="text-sm leading-6 text-ink-muted">
          Booking lapangan {booking.field_name || `#${booking.field_id}`} pada {formatDate(booking.booking_date)}, pukul {booking.start_time}–{booking.end_time} akan dibatalkan dan slotnya tersedia kembali.
        </p>
        {payment?.status === "PAID" && (
          <p className="mt-4 rounded-md bg-champagne-tint px-3 py-2 text-sm text-ink">
            Pembayaran yang sudah diterima tidak otomatis dikembalikan. Pengajuan refund perlu ditinjau secara manual.
          </p>
        )}
        <label htmlFor="cancellation-reason" className="mb-1.5 mt-5 block text-xs font-medium tracking-wide text-ink-muted">
          Alasan pembatalan (opsional)
        </label>
        <textarea
          id="cancellation-reason"
          value={cancelReason}
          onChange={(event) => setCancelReason(event.target.value)}
          maxLength={255}
          rows={3}
          className="w-full rounded-md border border-border bg-surface-raised px-4 py-3 text-sm text-ink focus:border-onyx focus:outline-none focus:ring-4 focus:ring-onyx/5"
          placeholder="Tulis alasan pembatalan"
          disabled={cancelling}
        />
        {cancelError && <p role="alert" className="mt-3 text-sm text-red-600">{cancelError}</p>}
      </Modal>
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

function isBookingInFuture(booking) {
  const startAt = new Date(`${booking.booking_date}T${booking.start_time}:00`);
  return !Number.isNaN(startAt.getTime()) && startAt.getTime() > Date.now();
}

function toETicketBooking(booking, user) {
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
    customer: {
      name: user?.name || "Pemesan",
      phone: user?.phone || "",
      email: user?.email || "",
    },
  };
}