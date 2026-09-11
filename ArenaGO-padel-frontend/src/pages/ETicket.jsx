import { useLocation, Link } from "react-router-dom";
import Icon from "../components/ui/Icon";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import StatusBadge from "../components/ui/StatusBadge";
import { sampleBooking } from "../lib/mockData";
import { formatDateLong, formatIDR } from "../lib/format";
import { bookingPriceBreakdown } from "../lib/mockData";
import BrandLogo from "../components/ui/BrandLogo";

export default function ETicket() {
  const { state } = useLocation();
  const booking = state?.booking ?? sampleBooking;
  const total = bookingPriceBreakdown(booking).find((b) => b.label === "__total__").amount;

  return (
    <div className="flex flex-col items-center px-5 py-10 sm:px-8">
      <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-sage-tint">
        <Icon name="check_circle" size={32} className="text-sage" filled />
      </div>
      <h1 className="mb-2 text-center font-display text-3xl">Booking Terkonfirmasi</h1>
      <p className="mb-8 max-w-sm text-center text-ink-muted">
        Tunjukkan QR ini kepada staf lapangan saat check-in. Detail juga sudah
        dikirim ke email Anda.
      </p>

      <Card className="w-full max-w-sm overflow-hidden !p-0">
        <div className="bg-onyx px-6 py-5 text-canvas">
          <div className="mb-1 flex items-center justify-between">
            <span className="text-canvas"><BrandLogo compact /></span>
            <StatusBadge status="confirmed" />
          </div>
          <p className="text-xs text-canvas/60">Booking ID · {booking.id}</p>
        </div>

        <div className="flex flex-col items-center gap-4 border-b border-dashed border-border px-6 py-8">
          <QrPlaceholder />
          <p className="text-xs text-ink-faint">Scan di meja resepsionis</p>
        </div>

        <div className="space-y-3 px-6 py-6 text-sm">
          <Row label="Lapangan" value={booking.court.name} />
          <Row label="Lokasi" value={booking.court.location} />
          <Row label="Tanggal" value={formatDateLong(booking.date)} />
          <Row label="Jam" value={`${booking.slots[0]} – ${booking.slots.length} jam`} />
          <Row label="Atas Nama" value={booking.customer.name} />
          <div className="flex items-baseline justify-between border-t border-border pt-3">
            <span className="font-medium">Total Dibayar</span>
            <span className="font-display text-xl">{formatIDR(total)}</span>
          </div>
        </div>
      </Card>

      <div className="mt-8 flex w-full max-w-sm flex-col gap-3">
        <Button variant="secondary" className="w-full" icon={<Icon name="download" size={18} />}>
          Unduh E-Tiket
        </Button>
        <Link to="/">
          <Button variant="ghost" className="w-full">
            Kembali ke Beranda
          </Button>
        </Link>
      </div>
    </div>
  );
}

function Row({ label, value }) {
  return (
    <div className="flex justify-between gap-4">
      <span className="text-ink-faint">{label}</span>
      <span className="text-right font-medium">{value}</span>
    </div>
  );
}

function QrPlaceholder() {
  const cells = Array.from({ length: 81 }).map((_, i) => (i * 13 + 5) % 6 === 0);
  return (
    <div className="grid h-44 w-44 grid-cols-9 gap-1 rounded-md border border-border bg-white p-3">
      {cells.map((filled, i) => (
        <span key={i} className={`rounded-[2px] ${filled ? "bg-onyx" : "bg-transparent"}`} />
      ))}
    </div>
  );
}
