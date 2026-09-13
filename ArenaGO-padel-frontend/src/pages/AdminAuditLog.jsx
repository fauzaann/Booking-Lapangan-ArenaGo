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
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    api.adminAuditLogs({ page, limit: 25 })
      .then((result) => {
        setLogs(result.data);
        setMeta(result.meta);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [page]);

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <div className="mb-8 flex flex-col justify-between gap-4 sm:flex-row sm:items-end">
        <div>
          <p className="mb-2 text-xs font-bold uppercase tracking-[0.12em] text-champagne">ArenaGO Admin</p>
          <h1 className="font-display text-3xl sm:text-4xl">Aktivitas sistem</h1>
          <p className="mt-2 text-ink-muted">Lihat seluruh request, aksi pengguna, dan aktivitas administrator.</p>
        </div>
        <button type="button" onClick={() => setPage(1)} className="inline-flex items-center gap-2 self-start rounded-full border border-border px-4 py-2.5 text-sm font-medium hover:border-ink-muted sm:self-auto">
          <Icon name="refresh" size={17} />
          Muat ulang
        </button>
      </div>

      {error ? <p className="rounded-md border border-terracotta/30 bg-terracotta/10 p-4 text-sm text-terracotta">{error}</p> : null}
      {loading ? <p className="py-16 text-center text-sm text-ink-muted">Memuat aktivitas...</p> : null}
      {!loading && !error && logs.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-20 text-center">
          <Icon name="history" size={32} className="mb-3 text-ink-faint" />
          <p className="mb-1 font-display text-xl">Belum ada aktivitas</p>
          <p className="text-sm text-ink-muted">Aktivitas baru akan muncul setelah ada request ke sistem.</p>
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