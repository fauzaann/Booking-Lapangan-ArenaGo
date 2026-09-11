import { useEffect, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import Card from "../components/ui/Card";
import Button from "../components/ui/Button";
import Icon from "../components/ui/Icon";
import StatusBadge from "../components/ui/StatusBadge";
import { api } from "../lib/api";
import { formatIDR } from "../lib/format";

const MAX_ATTEMPTS = 10;

export default function PaymentResult() {
  const location = useLocation();
  const failed = location.pathname.endsWith("/failed");
  const [booking, setBooking] = useState(null);
  const [payment, setPayment] = useState(null);
  const [loading, setLoading] = useState(!failed);
  const [error, setError] = useState("");
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (failed) {
      setLoading(false);
      return undefined;
    }

    let cancelled = false;
    let timer;
    const stored = JSON.parse(localStorage.getItem("arenago:pending-payment") || "null");
    if (!stored?.bookingId) {
      setError("Data booking tidak ditemukan. Silakan cek menu Riwayat.");
      setLoading(false);
      return undefined;
    }
    async function checkStatus() {
      try {
        const [bookingResult, paymentResult] = await Promise.all([
          api.booking(stored.bookingId),
          api.bookingPayment(stored.bookingId),
        ]);
        if (cancelled) return;
        setBooking(bookingResult.data);
        setPayment(paymentResult.data);
        if (paymentResult.data.status === "PAID" || bookingResult.data.status === "CONFIRMED") {
          localStorage.removeItem("arenago:pending-payment");
          setLoading(false);
          return;
        }
        if (attempt < MAX_ATTEMPTS) timer = window.setTimeout(() => setAttempt((value) => value + 1), 2000);
        else setLoading(false);
      } catch (requestError) {
        if (!cancelled) { setError(requestError.message); setLoading(false); }
      }
    }

    checkStatus();
    return () => { cancelled = true; window.clearTimeout(timer); };
  }, [attempt, failed]);

  if (failed) return <ResultMessage />;

  const paid = payment?.status === "PAID" || booking?.status === "CONFIRMED";
  return (
    <div className="flex min-h-[calc(100vh-160px)] items-center justify-center px-5 py-12 sm:px-8">
      <Card className="w-full max-w-lg text-center">
        <div className={`mx-auto mb-5 flex h-16 w-16 items-center justify-center rounded-full ${paid ? "bg-sage-tint text-sage" : "bg-champagne-tint text-champagne"}`}><Icon name={paid ? "check_circle" : "schedule"} size={34} filled={paid} /></div>
        <h1 className="mb-2 font-display text-3xl">{paid ? "Pembayaran Berhasil" : "Memverifikasi Pembayaran"}</h1>
        <p className="mb-6 text-sm text-ink-muted">{error || (paid ? "Booking Anda sudah terkonfirmasi." : loading ? "Kami sedang menunggu konfirmasi dari Xendit. Halaman ini akan diperbarui otomatis." : "Pembayaran belum terkonfirmasi. Jika sudah membayar, cek Riwayat Booking.")}</p>
        {booking && <div className="mb-6 space-y-3 rounded-md bg-surface px-5 py-4 text-left text-sm"><div className="flex justify-between gap-4"><span className="text-ink-faint">Booking</span><span className="font-medium">{booking.booking_code}</span></div><div className="flex justify-between gap-4"><span className="text-ink-faint">Lapangan</span><span className="font-medium">{booking.field_name || `Lapangan #${booking.field_id}`}</span></div><div className="flex justify-between gap-4"><span className="text-ink-faint">Total</span><span className="font-medium">{formatIDR(booking.total_price)}</span></div><div className="flex justify-between gap-4"><span className="text-ink-faint">Status</span><StatusBadge status={paid ? "confirmed" : "pending"} label={paid ? "Terkonfirmasi" : "Menunggu pembayaran"} /></div></div>}
        <div className="flex flex-col gap-3 sm:flex-row sm:justify-center"><Link to="/riwayat"><Button variant="secondary">Lihat Riwayat</Button></Link>{paid && <Link to="/"><Button>Kembali ke Beranda</Button></Link>}</div>
      </Card>
    </div>
  );
}

function ResultMessage() {
  return <div className="flex min-h-[calc(100vh-160px)] items-center justify-center px-5 py-12 sm:px-8"><Card className="w-full max-w-lg text-center"><div className="mx-auto mb-5 flex h-16 w-16 items-center justify-center rounded-full bg-terracotta/10 text-terracotta"><Icon name="error" size={34} /></div><h1 className="mb-2 font-display text-3xl">Pembayaran Tidak Berhasil</h1><p className="mb-6 text-sm text-ink-muted">Invoice belum dibayar atau pembayaran dibatalkan. Cek Riwayat Booking untuk status terbaru.</p><Link to="/riwayat"><Button>Lihat Riwayat Booking</Button></Link></Card></div>;
}
