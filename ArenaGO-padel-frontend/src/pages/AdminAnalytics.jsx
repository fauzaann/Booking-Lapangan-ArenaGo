import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import Card from "../components/ui/Card";
import Icon from "../components/ui/Icon";
import { api } from "../lib/api";
import { formatIDR } from "../lib/format";

const MONTHS = [
  "Jan", "Feb", "Mar", "Apr", "Mei", "Jun",
  "Jul", "Agu", "Sep", "Okt", "Nov", "Des",
];

export default function AdminAnalytics() {
  const [year, setYear] = useState(new Date().getFullYear());
  const [report, setReport] = useState(null);
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [error, setError] = useState("");
  const [exportError, setExportError] = useState("");

  useEffect(() => {
    let active = true;
    api.adminRevenueAnalytics(year)
      .then((result) => {
        if (active) setReport(result.data);
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
  }, [year]);

  const maxMonthlyRevenue = useMemo(
    () => Math.max(0, ...(report?.by_month || []).map((month) => Number(month.revenue) || 0)),
    [report],
  );
  const topVenue = report?.by_location?.[0];
  const topSlot = report?.best_selling_slots?.[0];

  async function exportPDF() {
    if (!report) return;
    setExporting(true);
    setExportError("");
    try {
      const { default: jsPDF } = await import("jspdf");
      const pdf = new jsPDF({ orientation: "portrait", unit: "pt", format: "a4" });
      pdf.setFont("helvetica", "bold");
      pdf.setFontSize(18);
      pdf.text("Laporan Analitik Pendapatan ArenaGO", 40, 48);
      pdf.setFont("helvetica", "normal");
      pdf.setFontSize(11);
      pdf.text(`Tahun ${report.year}`, 40, 68);
      pdf.text(`Total pendapatan: ${formatIDR(report.total_revenue)}`, 40, 88);
      pdf.text(`Booking dibayar: ${report.paid_bookings}`, 40, 106);

      let y = 138;
      y = addPdfSection(pdf, "Pendapatan per Bulan", (report.by_month || []).map((row) => ({
        label: `${MONTHS[row.month - 1] || row.month} - ${row.bookings} booking`,
        revenue: row.revenue,
      })), y);
      y = addPdfSection(pdf, "Pendapatan per Venue / Lokasi", (report.by_location || []).map((row) => ({
        label: `${row.name || "Lokasi tidak diisi"} - ${row.bookings} booking`,
        revenue: row.revenue,
      })), y);
      y = addPdfSection(pdf, "Pendapatan per Lapangan", (report.by_field || []).map((row) => ({
        label: `${row.name} (${row.location || "Lokasi tidak diisi"}) - ${row.bookings} booking`,
        revenue: row.revenue,
      })), y);
      addPdfSection(pdf, "Slot Jam Terlaris", (report.best_selling_slots || []).map((row) => ({
        label: `${row.start_time} - ${row.bookings} kali terisi`,
        revenue: row.revenue,
      })), y);

      pdf.save(`arenago-laporan-analitik-${report.year}.pdf`);
    } catch (exportFailure) {
      setExportError(`PDF gagal dibuat: ${exportFailure.message}`);
    } finally {
      setExporting(false);
    }
  }

  if (loading && !report) {
    return <div className="px-5 py-20 text-center text-sm text-ink-muted">Memuat analitik pendapatan...</div>;
  }
  if (error && !report) {
    return <div className="px-5 py-20 text-center text-sm text-red-600">{error}</div>;
  }

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <div className="mb-8 flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
        <div>
          <Link to="/admin" className="mb-3 inline-flex items-center gap-1 text-sm text-ink-muted hover:text-ink">
            <Icon name="arrow_back" size={16} />
            Dashboard admin
          </Link>
          <p className="mb-2 text-xs font-bold uppercase tracking-[0.12em] text-champagne">ArenaGO Admin</p>
          <h1 className="font-display text-3xl sm:text-4xl">Analitik pendapatan</h1>
          <p className="mt-2 text-ink-muted">Laporan dihitung dari transaksi berstatus dibayar pada tahun yang dipilih.</p>
        </div>
        <div className="flex flex-wrap items-end gap-3">
          <label className="text-sm font-medium">
            Tahun
            <select
              aria-label="Tahun laporan"
              value={year}
              onChange={(event) => {
                setReport(null);
                setLoading(true);
                setError("");
                setYear(Number(event.target.value));
              }}
              className="mt-1 block rounded-md border border-border bg-surface-raised px-3 py-2"
            >
              {Array.from({ length: 10 }, (_, index) => new Date().getFullYear() - index).map((value) => (
                <option key={value} value={value}>{value}</option>
              ))}
            </select>
          </label>
          <button
            type="button"
            onClick={exportPDF}
            disabled={!report || loading || exporting}
            className="inline-flex items-center gap-2 rounded-full bg-onyx px-4 py-2.5 text-sm font-medium text-canvas hover:bg-onyx-hover disabled:cursor-not-allowed disabled:opacity-60"
          >
            <Icon name="picture_as_pdf" size={17} />
            {exporting ? "Menyiapkan PDF..." : "Ekspor laporan PDF"}
          </button>
        </div>
      </div>

      {error && <p role="alert" className="mb-4 text-sm text-red-600">{error}</p>}
      {exportError && <p role="alert" className="mb-4 text-sm text-red-600">{exportError}</p>}

      <div className="mb-8 grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard label="Total pendapatan" value={formatIDR(report?.total_revenue || 0)} icon="payments" />
        <MetricCard label="Booking dibayar" value={report?.paid_bookings || 0} icon="event_available" />
        <MetricCard label="Venue teratas" value={topVenue?.name || "-"} detail={topVenue ? formatIDR(topVenue.revenue) : "Belum ada transaksi"} icon="location_on" />
        <MetricCard label="Slot jam terlaris" value={topSlot?.start_time || "-"} detail={topSlot ? `${topSlot.bookings} kali terisi` : "Belum ada transaksi"} icon="schedule" />
      </div>

      <Card className="mb-6">
        <div className="mb-6">
          <h2 className="font-display text-xl">Pendapatan per bulan</h2>
          <p className="mt-1 text-sm text-ink-muted">Hanya pembayaran berhasil; bulan tanpa transaksi tetap ditampilkan.</p>
        </div>
        <div className="grid min-h-48 grid-cols-6 items-end gap-2 sm:grid-cols-12 sm:gap-3">
          {(report?.by_month || []).map((month) => {
            const revenue = Number(month.revenue) || 0;
            const height = maxMonthlyRevenue > 0 ? Math.max((revenue / maxMonthlyRevenue) * 100, revenue > 0 ? 4 : 0) : 0;
            return (
              <div key={month.month} className="flex h-48 min-w-0 flex-col items-center justify-end gap-2" title={`${MONTHS[month.month - 1]}: ${formatIDR(revenue)}`}>
                <span className="max-w-full truncate text-[10px] text-ink-faint">{revenue > 0 ? formatIDR(revenue) : ""}</span>
                <div className="flex h-28 w-full items-end rounded-t bg-surface">
                  <div className="w-full rounded-t bg-sage transition-all" style={{ height: `${height}%` }} />
                </div>
                <span className="text-xs text-ink-muted">{MONTHS[month.month - 1]}</span>
              </div>
            );
          })}
        </div>
      </Card>

      <div className="mb-6 grid grid-cols-1 gap-6 xl:grid-cols-2">
        <RevenueTable
          title="Pendapatan per venue / lokasi"
          description="Venue dikelompokkan menggunakan lokasi yang tercatat pada lapangan."
          rows={report?.by_location || []}
          name={(row) => row.name || "Lokasi tidak diisi"}
        />
        <RevenueTable
          title="Pendapatan per lapangan"
          description="Perbandingan pendapatan dan jumlah booking dibayar."
          rows={report?.by_field || []}
          name={(row) => row.name}
          detail={(row) => row.location}
        />
      </div>

      <Card>
        <div className="mb-5">
          <h2 className="font-display text-xl">Slot jam terlaris</h2>
          <p className="mt-1 text-sm text-ink-muted">Diurutkan berdasarkan berapa kali setiap slot satu jam terisi pada booking yang dibayar.</p>
        </div>
        {!report?.best_selling_slots?.length ? (
          <EmptyReport />
        ) : (
          <div className="divide-y divide-border">
            {report.best_selling_slots.map((slot, index) => (
              <div key={slot.start_time} className="flex items-center justify-between gap-4 py-3 first:pt-0">
                <div className="flex items-center gap-3">
                  <span className="flex h-8 w-8 items-center justify-center rounded-full bg-sage-tint text-sm font-semibold text-sage">{index + 1}</span>
                  <div>
                    <p className="font-medium">{slot.start_time}</p>
                    <p className="text-xs text-ink-muted">{slot.bookings} kali terisi</p>
                  </div>
                </div>
                <span className="text-sm font-semibold">{formatIDR(slot.revenue)}</span>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}

function MetricCard({ label, value, detail, icon }) {
  return (
    <Card padded className="p-5">
      <div className="mb-5 flex items-start justify-between">
        <p className="max-w-[12rem] text-xs font-bold uppercase tracking-[0.1em] text-ink-faint">{label}</p>
        <span className="flex h-9 w-9 items-center justify-center rounded-full bg-sage-tint text-sage"><Icon name={icon} size={18} /></span>
      </div>
      <p className="mb-1 truncate font-display text-2xl">{value}</p>
      {detail && <p className="truncate text-xs text-ink-muted">{detail}</p>}
    </Card>
  );
}

function RevenueTable({ title, description, rows, name, detail }) {
  return (
    <Card>
      <div className="mb-5">
        <h2 className="font-display text-xl">{title}</h2>
        <p className="mt-1 text-sm text-ink-muted">{description}</p>
      </div>
      {rows.length === 0 ? (
        <EmptyReport />
      ) : (
        <div className="divide-y divide-border">
          {rows.map((row) => (
            <div key={`${row.id || row.name}-${row.location || ""}`} className="flex items-center justify-between gap-4 py-3 first:pt-0">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{name(row)}</p>
                {detail?.(row) && <p className="mt-0.5 truncate text-xs text-ink-muted">{detail(row)}</p>}
                <p className="mt-0.5 text-xs text-ink-faint">{row.bookings} booking dibayar</p>
              </div>
              <span className="shrink-0 text-sm font-semibold">{formatIDR(row.revenue)}</span>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

function EmptyReport() {
  return <p className="py-8 text-center text-sm text-ink-faint">Belum ada transaksi dibayar pada periode ini.</p>;
}

function addPdfSection(pdf, title, rows, startY) {
  let y = startY;
  if (y > 740) {
    pdf.addPage();
    y = 45;
  }
  pdf.setFont("helvetica", "bold");
  pdf.setFontSize(13);
  pdf.text(title, 40, y);
  y += 20;
  pdf.setFont("helvetica", "normal");
  pdf.setFontSize(9);
  if (rows.length === 0) rows = [{ label: "Tidak ada transaksi", revenue: 0 }];
  for (const row of rows) {
    const lines = pdf.splitTextToSize(row.label, 360);
    if (y + lines.length * 12 > 790) {
      pdf.addPage();
      y = 45;
    }
    pdf.text(lines, 40, y);
    pdf.text(formatIDR(row.revenue), 570, y, { align: "right" });
    y += Math.max(16, lines.length * 12);
  }
  return y + 18;
}
