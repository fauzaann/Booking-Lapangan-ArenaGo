import { useEffect, useState } from "react";
import { useLocation, Link, useParams } from "react-router-dom";
import Icon from "../components/ui/Icon";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import StatusBadge from "../components/ui/StatusBadge";
import { formatIDR } from "../lib/format";
import BrandLogo from "../components/ui/BrandLogo";
import { api } from "../lib/api";
import QRCode from "qrcode";

// Fungsi format tanggal khusus untuk PDF (tidak bergantung browser locale)
function formatDateForPdf(date) {
  if (!date) return "-";
  const d = date instanceof Date ? date : new Date(`${date}T00:00:00`);
  const days = ["Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"];
  const months = ["Januari", "Februari", "Maret", "April", "Mei", "Juni",
    "Juli", "Agustus", "September", "Oktober", "November", "Desember"];
  return `${days[d.getDay()]}, ${d.getDate()} ${months[d.getMonth()]} ${d.getFullYear()}`;
}

export default function ETicket() {
  const { state } = useLocation();
  const { id } = useParams();
  const [booking, setBooking] = useState(null);
  const [ticketQr, setTicketQr] = useState("");
  const [downloading, setDownloading] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    api.booking(id)
      .then(({ data }) => {
        if (data.status !== "CONFIRMED" || data.payment?.status !== "PAID" || !data.check_in_token) {
          throw new Error("E-tiket tersedia setelah pembayaran berhasil dikonfirmasi.");
        }
        if (active) {
          setBooking({
            ...data,
            court: {
              name: data.field_name || `Lapangan #${data.field_id}`,
              location: state?.booking?.court?.location || "-",
            },
            date: new Date(`${data.booking_date}T00:00:00`),
            slots: [data.start_time, data.end_time],
            customer: state?.booking?.customer || { name: "Pemesan" },
          });
        }
      })
      .catch((requestError) => {
        if (active) setError(requestError.message);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [id, state]);

  useEffect(() => {
    if (!booking?.check_in_token || booking.checked_in_at) {
      return undefined;
    }

    let active = true;
    const payload = JSON.stringify({
      type: "ARENAGO_CHECK_IN",
      booking_code: booking.booking_code,
      check_in_token: booking.check_in_token,
    });
    QRCode.toDataURL(payload, { errorCorrectionLevel: "H", margin: 2, width: 320 })
      .then((dataUrl) => {
        if (active) setTicketQr(dataUrl);
      })
      .catch((qrError) => {
        if (active) setError(`Gagal membuat QR e-tiket: ${qrError.message}`);
      });
    return () => {
      active = false;
    };
  }, [booking]);

  if (loading) return <div className="px-5 py-20 text-center text-sm text-ink-muted">Memuat e-tiket...</div>;
  if (error || !booking) return <div role="alert" className="px-5 py-20 text-center text-sm text-red-600">{error || "E-tiket tidak ditemukan."}</div>;

  const total = booking.total_price;

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
      pdf.text(`Booking ID · ${booking.booking_code || "-"}`, cardX + 18, y + 56);

      y += 78;

      // ── QR Code section (white bg) ───────────────────────────────────
      const qrSectionH = 160;
      pdf.setFillColor(250, 250, 250);
      pdf.setDrawColor(220, 220, 220);
      pdf.rect(cardX, y, cardW, qrSectionH, "FD");

      const qrSize = 110;
      const qrX = cardX + (cardW - qrSize) / 2;
      const qrY = y + 20;
      if (booking.checked_in_at) {
        pdf.setFont("helvetica", "bold");
        pdf.setFontSize(11);
        pdf.setTextColor(74, 140, 100);
        pdf.text("TIKET SUDAH DIGUNAKAN", W / 2, qrY + 55, { align: "center" });
      } else {
        pdf.addImage(ticketQr, "PNG", qrX, qrY, qrSize, qrSize);
      }

      // QR caption
      pdf.setFontSize(8);
      pdf.setFont("helvetica", "normal");
      pdf.setTextColor(120, 120, 120);
      pdf.text(
        booking.checked_in_at ? "Check-in sudah tercatat" : "Pindai oleh admin saat check-in",
        W / 2,
        qrY + qrSize + 18,
        { align: "center" }
      );

      y += qrSectionH;

      // ── Detail rows (white body) ─────────────────────────────────────
      const rows = [
        ["Lapangan", booking.court?.name || "-"],
        ["Lokasi", booking.court?.location || "-"],
        ["Tanggal", formatDateForPdf(booking.booking_date)],
        ["Jam", `${booking.start_time} – ${booking.end_time}`],
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

      pdf.save(`ETicket-${booking.booking_code || "ArenaGo"}.pdf`);
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
        Tunjukkan QR ini kepada admin saat check-in. QR hanya dapat digunakan satu kali.
      </p>

      <Card className="w-full max-w-sm overflow-hidden !p-0">
        <div className="bg-onyx px-6 py-5 text-canvas">
          <div className="mb-1 flex items-center justify-between">
            <span className="text-canvas"><BrandLogo compact /></span>
            <StatusBadge status="confirmed" />
          </div>
          <p className="text-xs text-canvas/60">Booking ID · {booking.booking_code}</p>
        </div>

        <div className="flex flex-col items-center gap-4 border-b border-dashed border-border px-6 py-8">
          {booking.checked_in_at ? (
            <p className="flex h-44 items-center gap-2 text-sm font-medium text-sage">
              <Icon name="verified" size={22} />
              Tiket sudah digunakan
            </p>
          ) : (
            <>
              {ticketQr ? (
                <img src={ticketQr} alt="QR e-tiket untuk check-in" className="h-44 w-44 rounded-md border border-border bg-white p-2" />
              ) : (
                <div className="h-44 w-44 animate-pulse rounded-md bg-surface" />
              )}
              <p className="text-xs text-ink-faint">Pindai oleh admin saat check-in</p>
            </>
          )}
        </div>

        <div className="space-y-3 px-6 py-6 text-sm">
          <Row label="Lapangan" value={booking.court.name} />
          <Row label="Lokasi" value={booking.court.location} />
          <Row label="Tanggal" value={formatDateForPdf(booking.booking_date)} />
          <Row label="Jam" value={`${booking.start_time} – ${booking.end_time}`} />
          <Row label="Atas Nama" value={booking.customer?.name || "Pemesan"} />
          <Row label="Status check-in" value={booking.checked_in_at ? "Sudah check-in" : "Belum check-in"} />
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
          disabled={downloading || (!ticketQr && !booking.checked_in_at)}
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
