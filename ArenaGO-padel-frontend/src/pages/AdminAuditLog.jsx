import { useEffect, useState } from "react";
import Icon from "../components/ui/Icon";
import { api } from "../lib/api";

function formatTimestamp(value) {
  return new Intl.DateTimeFormat("id-ID", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}

function statusClass(status) {
  if (status >= 500) return "bg-terracotta/15 text-terracotta";
  if (status >= 400) return "bg-champagne/30 text-ink";
  return "bg-sage-tint text-sage";
}

export default function AdminAuditLog() {
  const [logs, setLogs] = useState([]);
  const [meta, setMeta] = useState(null);
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState({});
  const [filterDraft, setFilterDraft] = useState({
    actor_email: "",
    method: "",
    path: "",
    status_code: "",
    from: "",
    to: "",
  });
  const [refreshKey, setRefreshKey] = useState(0);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const hasFilters = Object.keys(filters).length > 0;

  useEffect(() => {
    setLoading(true);
    setError("");
    api.adminAuditLogs({ page, limit: 25, ...filters })
      .then((result) => {
        setLogs(result.data);
        setMeta(result.meta);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [page, filters, refreshKey]);

  function applyFilters(event) {
    event.preventDefault();
    setPage(1);
    setFilters(Object.fromEntries(
      Object.entries(filterDraft).filter(([, value]) => value !== ""),
    ));
  }

  function clearFilters() {
    setFilterDraft({
      actor_email: "",
      method: "",
      path: "",
      status_code: "",
      from: "",
      to: "",
    });
    setFilters({});
    setPage(1);
  }

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <div className="mb-8 flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
        <div>
          <p className="mb-2 text-xs font-bold uppercase tracking-[0.12em] text-champagne">ArenaGO Admin</p>
          <h1 className="font-display text-3xl sm:text-4xl">Aktivitas sistem</h1>
          <p className="mt-2 text-ink-muted">Lihat seluruh request, aksi pengguna, dan aktivitas administrator.</p>
        </div>
        <button type="button" onClick={() => setRefreshKey((key) => key + 1)} className="inline-flex items-center gap-2 self-start rounded-full border border-border px-4 py-2.5 text-sm font-medium hover:border-ink-muted sm:self-auto">
          <Icon name="refresh" size={17} />
          Muat ulang
        </button>
      </div>

      <form onSubmit={applyFilters} className="mb-6 rounded-lg border border-border bg-surface-raised p-4 sm:p-5">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <label className="text-sm font-medium">
            Email pengguna
            <input
              type="search"
              value={filterDraft.actor_email}
              onChange={(event) => setFilterDraft((current) => ({ ...current, actor_email: event.target.value }))}
              placeholder="Cari email pengguna"
              className="mt-1.5 h-11 w-full rounded-md border border-border bg-surface px-3 text-sm font-normal focus:border-onyx focus:outline-none focus:ring-4 focus:ring-onyx/5"
            />
          </label>
          <label className="text-sm font-medium">
            Aktivitas / endpoint
            <input
              type="search"
              value={filterDraft.path}
              onChange={(event) => setFilterDraft((current) => ({ ...current, path: event.target.value }))}
              placeholder="Contoh: bookings, payments"
              className="mt-1.5 h-11 w-full rounded-md border border-border bg-surface px-3 text-sm font-normal focus:border-onyx focus:outline-none focus:ring-4 focus:ring-onyx/5"
            />
          </label>
          <label className="text-sm font-medium">
            Metode
            <select
              value={filterDraft.method}
              onChange={(event) => setFilterDraft((current) => ({ ...current, method: event.target.value }))}
              className="mt-1.5 h-11 w-full rounded-md border border-border bg-surface px-3 text-sm font-normal focus:border-onyx focus:outline-none focus:ring-4 focus:ring-onyx/5"
            >
              <option value="">Semua metode</option>
              {["GET", "POST", "PUT", "PATCH", "DELETE"].map((method) => (
                <option key={method} value={method}>{method}</option>
              ))}
            </select>
          </label>
          <label className="text-sm font-medium">
            Kode status HTTP
            <input
              type="number"
              min="100"
              max="599"
              value={filterDraft.status_code}
              onChange={(event) => setFilterDraft((current) => ({ ...current, status_code: event.target.value }))}
              placeholder="Contoh: 200 atau 500"
              className="mt-1.5 h-11 w-full rounded-md border border-border bg-surface px-3 text-sm font-normal focus:border-onyx focus:outline-none focus:ring-4 focus:ring-onyx/5"
            />
          </label>
          <label className="text-sm font-medium">
            Dari tanggal
            <input
              type="date"
              value={filterDraft.from}
              onChange={(event) => setFilterDraft((current) => ({ ...current, from: event.target.value }))}
              className="mt-1.5 h-11 w-full rounded-md border border-border bg-surface px-3 text-sm font-normal focus:border-onyx focus:outline-none focus:ring-4 focus:ring-onyx/5"
            />
          </label>
          <label className="text-sm font-medium">
            Sampai tanggal
            <input
              type="date"
              value={filterDraft.to}
              onChange={(event) => setFilterDraft((current) => ({ ...current, to: event.target.value }))}
              className="mt-1.5 h-11 w-full rounded-md border border-border bg-surface px-3 text-sm font-normal focus:border-onyx focus:outline-none focus:ring-4 focus:ring-onyx/5"
            />
          </label>
        </div>
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <button type="submit" className="rounded-full bg-onyx px-5 py-2.5 text-sm font-semibold text-white hover:bg-ink">
            Terapkan filter
          </button>
          <button type="button" onClick={clearFilters} className="rounded-full border border-border px-5 py-2.5 text-sm font-medium hover:border-ink-muted">
            Hapus filter
          </button>
          {meta ? <span className="text-sm text-ink-muted">{meta.total_items} aktivitas ditemukan</span> : null}
        </div>
      </form>

      {error ? <p className="rounded-md border border-terracotta/30 bg-terracotta/10 p-4 text-sm text-terracotta">{error}</p> : null}
      {loading ? <p className="py-16 text-center text-sm text-ink-muted">Memuat aktivitas...</p> : null}
      {!loading && !error && logs.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-20 text-center">
          <Icon name="history" size={32} className="mb-3 text-ink-faint" />
          <p className="mb-1 font-display text-xl">{hasFilters ? "Tidak ada aktivitas yang cocok" : "Belum ada aktivitas"}</p>
          <p className="text-sm text-ink-muted">
            {hasFilters ? "Coba ubah atau hapus filter pencarian." : "Aktivitas baru akan muncul setelah ada request ke sistem."}
          </p>
        </div>
      ) : null}
      {!loading && !error && logs.length > 0 ? (
        <>
          <div className="overflow-hidden rounded-lg border border-border">
            <div className="hidden grid-cols-[1.1fr_1.5fr_0.7fr_0.7fr_1fr] gap-4 border-b border-border bg-surface px-6 py-3 text-xs font-semibold uppercase tracking-wide text-ink-faint md:grid">
              <span>Waktu / aktor</span>
              <span>Request</span>
              <span>Status</span>
              <span>Durasi</span>
              <span>IP address</span>
            </div>
            {logs.map((log) => (
              <div key={log.id} className="grid gap-3 border-b border-border px-5 py-4 last:border-b-0 hover:bg-surface/60 md:grid-cols-[1.1fr_1.5fr_0.7fr_0.7fr_1fr] md:items-center md:gap-4 md:px-6">
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold">{log.actor_email || "Sistem / publik"}</p>
                  <p className="mt-1 text-xs text-ink-faint">{formatTimestamp(log.created_at)}</p>
                </div>
                <div className="min-w-0">
                  <p className="truncate font-mono text-xs font-semibold">{log.method} {log.path}</p>
                  <p className="mt-1 truncate text-xs text-ink-faint">{log.user_agent || "User agent tidak tersedia"}</p>
                </div>
                <span className={`w-fit rounded-full px-2.5 py-1 text-xs font-semibold ${statusClass(log.status_code)}`}>{log.status_code}</span>
                <p className="text-sm text-ink-muted">{log.duration_ms} ms</p>
                <p className="font-mono text-xs text-ink-muted">{log.ip_address || "-"}</p>
              </div>
            ))}
          </div>
          {meta && meta.total_pages > 1 ? (
            <div className="mt-5 flex items-center justify-between text-sm">
              <span className="text-ink-muted">Halaman {meta.page} dari {meta.total_pages}</span>
              <div className="flex gap-2">
                <button type="button" disabled={page <= 1} onClick={() => setPage((current) => current - 1)} className="rounded-full border border-border px-3 py-2 disabled:cursor-not-allowed disabled:opacity-40">Sebelumnya</button>
                <button type="button" disabled={page >= meta.total_pages} onClick={() => setPage((current) => current + 1)} className="rounded-full border border-border px-3 py-2 disabled:cursor-not-allowed disabled:opacity-40">Berikutnya</button>
              </div>
            </div>
          ) : null}
        </>
      ) : null}
    </div>
  );
}