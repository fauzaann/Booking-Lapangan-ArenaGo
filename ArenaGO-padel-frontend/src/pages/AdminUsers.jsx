import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { useAuth } from "../context/AuthContext";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import Icon from "../components/ui/Icon";
import { api } from "../lib/api";

const PAGE_SIZE = 20;

export default function AdminUsers() {
  const { user: currentUser } = useAuth();
  const [users, setUsers] = useState([]);
  const [meta, setMeta] = useState(null);
  const [page, setPage] = useState(1);
  const [refreshKey, setRefreshKey] = useState(0);
  const [loadedQuery, setLoadedQuery] = useState(null);
  const [updatingID, setUpdatingID] = useState(null);
  const [error, setError] = useState("");
  const [feedback, setFeedback] = useState("");
  const queryKey = `${page}:${refreshKey}`;
  const loading = loadedQuery !== queryKey;

  useEffect(() => {
    let active = true;
    api.adminUsers({ page: String(page), limit: String(PAGE_SIZE) })
      .then((result) => {
        if (!active) return;
        setUsers(result.data);
        setMeta(result.meta);
      })
      .catch((err) => {
        if (active) setError(err.message || "Gagal memuat data pengguna.");
      })
      .finally(() => {
        if (active) setLoadedQuery(queryKey);
      });
    return () => {
      active = false;
    };
  }, [page, queryKey, refreshKey]);

  async function updateStatus(account) {
    setUpdatingID(account.id);
    setError("");
    setFeedback("");
    try {
      const result = await api.updateAdminUserStatus(account.id, !account.is_active);
      setUsers((current) => current.map((item) => item.id === account.id ? result.data : item));
      setFeedback(`${account.name} sekarang ${result.data.is_active ? "aktif" : "nonaktif"}.`);
    } catch (err) {
      setError(err.message || "Gagal mengubah status pengguna.");
    } finally {
      setUpdatingID(null);
    }
  }

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <Link to="/admin" className="mb-6 inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink">
        <Icon name="arrow_back" size={18} />
        Dashboard admin
      </Link>
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="mb-2 text-xs font-bold uppercase tracking-[0.12em] text-champagne">ArenaGO Admin</p>
          <h1 className="font-display text-3xl sm:text-4xl">Pemantauan pengguna</h1>
          <p className="mt-2 text-sm text-ink-muted">Lihat nama, email, role, dan status akun pengguna.</p>
        </div>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => {
            setError("");
            setRefreshKey((key) => key + 1);
          }}
          disabled={loading}
        >
          <Icon name="refresh" size={17} />
          Muat ulang
        </Button>
      </div>

      <p className="mb-4 text-xs text-ink-faint">Akun admin dan akun yang sedang digunakan tidak dapat dinonaktifkan.</p>
      {error ? <p role="alert" className="mb-4 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700">{error}</p> : null}
      {feedback ? <p role="status" className="mb-4 rounded-md bg-emerald-50 px-4 py-3 text-sm text-emerald-800">{feedback}</p> : null}

      {loading ? (
        <p className="py-16 text-center text-sm text-ink-muted">Memuat pengguna...</p>
      ) : users.length === 0 ? (
        <Card className="py-12 text-center text-sm text-ink-muted">Belum ada akun pengguna.</Card>
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <div className="hidden grid-cols-[1.4fr_1.2fr_0.7fr_0.9fr_0.9fr] gap-4 border-b border-border bg-surface px-6 py-3 text-xs font-semibold uppercase tracking-wide text-ink-faint md:grid">
            <span>Nama pengguna</span>
            <span>Email</span>
            <span>Role</span>
            <span>Status</span>
            <span className="text-right">Aksi</span>
          </div>
          {users.map((account) => {
            const isSelf = String(account.id) === String(currentUser?.id);
            const isAdmin = account.role?.toUpperCase() === "ADMIN";
            const cannotDeactivate = account.is_active && (isAdmin || isSelf);
            return (
              <div key={account.id} className="grid gap-3 border-b border-border px-5 py-4 last:border-b-0 md:grid-cols-[1.4fr_1.2fr_0.7fr_0.9fr_0.9fr] md:items-center md:gap-4 md:px-6">
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold">{account.name}</p>
                  <p className="mt-1 text-xs text-ink-faint">Terdaftar {formatDate(account.created_at)}</p>
                </div>
                <p className="break-all text-sm text-ink-muted">{account.email}</p>
                <span className="w-fit rounded-full bg-surface px-2.5 py-1 text-xs font-semibold text-ink-muted">
                  {isAdmin ? "Admin" : "User"}
                </span>
                <span className={`w-fit rounded-full px-2.5 py-1 text-xs font-semibold ${account.is_active ? "bg-sage-tint text-sage" : "bg-terracotta/10 text-terracotta"}`}>
                  {account.is_active ? "Aktif" : "Nonaktif"}
                </span>
                <div className="flex justify-start md:justify-end">
                  <Button
                    variant={account.is_active ? "danger" : "secondary"}
                    size="sm"
                    disabled={updatingID === account.id || cannotDeactivate}
                    title={cannotDeactivate ? "Akun ini tidak dapat dinonaktifkan" : undefined}
                    onClick={() => updateStatus(account)}
                  >
                    {updatingID === account.id ? "Menyimpan..." : account.is_active ? "Nonaktifkan" : "Aktifkan"}
                  </Button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {meta?.total_pages > 1 ? (
        <div className="mt-5 flex items-center justify-between">
          <Button variant="secondary" size="sm" disabled={page <= 1 || loading} onClick={() => {
            setError("");
            setPage((value) => value - 1);
          }}>
            Sebelumnya
          </Button>
          <span className="text-sm text-ink-muted">Halaman {page} dari {meta.total_pages} · {meta.total_items} akun</span>
          <Button variant="secondary" size="sm" disabled={page >= meta.total_pages || loading} onClick={() => {
            setError("");
            setPage((value) => value + 1);
          }}>
            Berikutnya
          </Button>
        </div>
      ) : meta ? (
        <p className="mt-4 text-right text-sm text-ink-muted">{meta.total_items} akun</p>
      ) : null}
    </div>
  );
}

function formatDate(value) {
  if (!value) return "-";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return "-";
  return new Intl.DateTimeFormat("id-ID", { dateStyle: "medium" }).format(parsed);
}
