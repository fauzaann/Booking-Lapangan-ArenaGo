import { useState } from "react";
import { useLocation, Link } from "react-router-dom";
import Icon from "../components/ui/Icon";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import StatusBadge from "../components/ui/StatusBadge";
import { sampleBooking } from "../lib/mockData";
import { formatDateLong, formatIDR } from "../lib/format";
import { bookingPriceBreakdown } from "../lib/mockData";
import BrandLogo from "../components/ui/BrandLogo";

// Fungsi format tanggal khusus untuk PDF (tidak bergantung browser locale)
function formatDateForPdf(date) {
  if (!date) return "-";
  const d = date instanceof Date ? date : new Date(date);
  const days = ["Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"];
  const months = ["Januari", "Februari", "Maret", "April", "Mei", "Juni",
    "Juli", "Agustus", "September", "Oktober", "November", "Desember"];
  return `${days[d.getDay()]}, ${d.getDate()} ${months[d.getMonth()]} ${d.getFullYear()}`;
}

// Generate QR cells dari bookingId untuk PDF
function getQrCells(bookingId) {
  const seed = bookingId
    ? bookingId.split("").reduce((acc, c) => acc + c.charCodeAt(0), 0)
    : 42;
  return Array.from({ length: 81 }).map((_, i) => (i * seed + 5) % 7 < 3);
}

export default function ETicket() {
  const { state } = useLocation();
  const booking = state?.booking ?? sampleBooking;
  const total = bookingPriceBreakdown(booking).find((b) => b.label === "__total__").amount;
  const [downloading, setDownloading] = useState(false);

  async function downloadPdf() {
    setDownloading(true);
    try {
      const { default: jsPDF } = await import("jspdf");

      const pdf = new jsPDF({ orientation: "portrait", unit: "pt", format: "a4" });

      const W = 595.28; // A4 width in pt
      const cardW = 340;
      const cardX = (W - cardW) / 2;
      let y = 50;

      // ── Header: dark background ──────────────────────────────────────
      pdf.setFillColor(20, 23, 26);         // onyx #14171A
      pdf.roundedRect(cardX, y, cardW, 70, 8, 8, "F");

      // Brand name
      pdf.setFont("helvetica", "bold");
      pdf.setFontSize(16);
      pdf.setTextColor(255, 255, 255);
      pdf.text("ArenaGo", cardX + 18, y + 26);

      // "Terkonfirmasi" badge
      pdf.setFillColor(74, 180, 120);       // sage green
      pdf.roundedRect(cardX + cardW - 108, y + 12, 96, 20, 4, 4, "F");
      pdf.setFontSize(8.5);
      pdf.setFont("helvetica", "bold");
      pdf.setTextColor(255, 255, 255);
      pdf.text("TERKONFIRMASI", cardX + cardW - 106, y + 26);

      // Booking ID
      pdf.setFont("helvetica", "normal");
      pdf.setFontSize(8);
      pdf.setTextColor(160, 160, 160);
      pdf.text(`Booking ID · ${booking.id || "-"}`, cardX + 18, y + 56);

      y += 78;

      // ── QR Code section (white bg) ───────────────────────────────────
      const qrSectionH = 160;
      pdf.setFillColor(250, 250, 250);
      pdf.setDrawColor(220, 220, 220);
      pdf.rect(cardX, y, cardW, qrSectionH, "FD");

      // Draw QR grid
      const qrCells = getQrCells(booking.id);
      const qrSize = 100;
      const qrX = cardX + (cardW - qrSize) / 2;
      const qrY = y + 20;
      const cellSize = qrSize / 9;

      // QR background
      pdf.setFillColor(255, 255, 255);
      pdf.rect(qrX - 4, qrY - 4, qrSize + 8, qrSize + 8, "F");

      qrCells.forEach((filled, i) => {
        const col = i % 9;
        const row = Math.floor(i / 9);
        if (filled) {
          pdf.setFillColor(20, 23, 26); // onyx
        } else {
          pdf.setFillColor(255, 255, 255); // white
        }
        pdf.rect(
          qrX + col * cellSize,
          qrY + row * cellSize,
          cellSize - 0.5,
          cellSize - 0.5,
          "F"
        );
      });

      // QR caption
      pdf.setFontSize(8);
      pdf.setFont("helvetica", "normal");
      pdf.setTextColor(120, 120, 120);
      pdf.text("Scan di meja resepsionis", W / 2, qrY + qrSize + 18, { align: "center" });

      y += qrSectionH;

      // ── Detail rows (white body) ─────────────────────────────────────
      const rows = [
        ["Lapangan", booking.court?.name || "-"],
        ["Lokasi", booking.court?.location || "-"],
        ["Tanggal", formatDateForPdf(booking.date)],
        ["Jam", `${booking.slots?.[0] || "-"} – ${(booking.slots?.length || 1)} jam`],
        ["Atas Nama", booking.customer?.name || "Pemesan"],
        booking.customer?.phone ? ["WhatsApp", booking.customer.phone] : null,
        booking.customer?.email ? ["Email", booking.customer.email] : null,
      ].filter(Boolean);

      const rowH = 28;
      const detailH = rows.length * rowH + 50; // +50 for total row

      pdf.setFillColor(255, 255, 255);
      pdf.setDrawColor(220, 220, 220);
      pdf.rect(cardX, y, cardW, detailH, "FD");

      rows.forEach(([label, value], idx) => {
        const rowY = y + 18 + idx * rowH;
        // Separator line
        if (idx > 0) {
          pdf.setDrawColor(235, 235, 235);
          pdf.line(cardX + 14, rowY - 8, cardX + cardW - 14, rowY - 8);
        }
        pdf.setFont("helvetica", "normal");
        pdf.setFontSize(8.5);
        pdf.setTextColor(130, 130, 130);
        pdf.text(label, cardX + 18, rowY);

        pdf.setFont("helvetica", "bold");
        pdf.setFontSize(8.5);
        pdf.setTextColor(30, 30, 30);
        // Truncate long values
        const maxChars = 36;
        const display = String(value).length > maxChars ? String(value).slice(0, maxChars) + "…" : String(value);
        pdf.text(display, cardX + cardW - 18, rowY, { align: "right" });
      });

      // Total divider
      const totalY = y + detailH - 38;
      pdf.setDrawColor(180, 180, 180);
      pdf.line(cardX + 14, totalY, cardX + cardW - 14, totalY);

      pdf.setFont("helvetica", "bold");
      pdf.setFontSize(9);
      pdf.setTextColor(30, 30, 30);
      pdf.text("Total Dibayar", cardX + 18, totalY + 16);

      pdf.setFont("helvetica", "bold");
      pdf.setFontSize(13);
      pdf.setTextColor(20, 23, 26);
      pdf.text(formatIDR(total), cardX + cardW - 18, totalY + 17, { align: "right" });

      y += detailH;

      // ── Footer ───────────────────────────────────────────────────────
      pdf.setFont("helvetica", "normal");
      pdf.setFontSize(7.5);
      pdf.setTextColor(150, 150, 150);
      pdf.text("ArenaGo · Booking Lapangan Padel · Dokumen ini adalah e-tiket resmi.", W / 2, y + 20, { align: "center" });

      pdf.save(`ETicket-${booking.id || "ArenaGo"}.pdf`);
    } catch (err) {
      console.error("PDF generation failed:", err);
      // Fallback: print dialog
      window.print();
    } finally {
      setDownloading(false);
    }
  }

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
          <QrPlaceholder bookingId={booking.id} />
          <p className="text-xs text-ink-faint">Scan di meja resepsionis</p>
        </div>

        <div className="space-y-3 px-6 py-6 text-sm">
          <Row label="Lapangan" value={booking.court.name} />
          <Row label="Lokasi" value={booking.court.location} />
          <Row label="Tanggal" value={formatDateLong(booking.date)} />
          <Row label="Jam" value={`${booking.slots[0]} – ${booking.slots.length} jam`} />
          <Row label="Atas Nama" value={booking.customer?.name || "Pemesan"} />
          {booking.customer?.phone && (
            <Row label="WhatsApp" value={booking.customer.phone} />
          )}
          <div className="flex items-baseline justify-between border-t border-border pt-3">
            <span className="font-medium">Total Dibayar</span>
            <span className="font-display text-xl">{formatIDR(total)}</span>
          </div>
        </div>
      </Card>

      <div className="mt-8 flex w-full max-w-sm flex-col gap-3">
        <Button
          className="w-full"
          onClick={downloadPdf}
          disabled={downloading}
          icon={<Icon name={downloading ? "hourglass_top" : "download"} size={18} />}
        >
          {downloading ? "Menyiapkan PDF..." : "Unduh E-Tiket (PDF)"}
        </Button>
        <Button
          variant="secondary"
          className="w-full"
          onClick={() => window.print()}
          icon={<Icon name="print" size={18} />}
        >
          Cetak
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

function QrPlaceholder({ bookingId }) {
  const cells = getQrCells(bookingId);
  return (
    <div className="grid h-44 w-44 grid-cols-9 gap-1 rounded-md border border-border bg-white p-3">
      {cells.map((filled, i) => (
        <span key={i} className={`rounded-[2px] ${filled ? "bg-onyx" : "bg-transparent"}`} />
      ))}
    </div>
  );
}
