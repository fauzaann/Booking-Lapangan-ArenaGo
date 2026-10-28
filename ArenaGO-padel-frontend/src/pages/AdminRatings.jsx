import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import Icon from "../components/ui/Icon";
import { api } from "../lib/api";

const PAGE_SIZE = 20;

export default function AdminRatings() {
  const [ratings, setRatings] = useState([]);
  const [meta, setMeta] = useState(null);
  const [visibility, setVisibility] = useState("");
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [updatingID, setUpdatingID] = useState(null);
  const [error, setError] = useState("");
  const [feedback, setFeedback] = useState("");

  useEffect(() => {
    let active = true;
    const params = { page: String(page), limit: String(PAGE_SIZE) };
    if (visibility !== "") params.is_visible = visibility;
    api.adminRatings(params)
      .then((result) => {
        if (!active) return;
        setRatings(result.data);
        setMeta(result.meta);
      })
      .catch((err) => {
        if (active) setError(err.message);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [page, visibility]);

  const moderate = async (rating) => {
    setUpdatingID(rating.id);
    setError("");
    setFeedback("");
    try {
      await api.moderateRating(rating.id, !rating.is_visible);
      setRatings((current) => visibility === ""
        ? current.map((item) => item.id === rating.id ? { ...item, is_visible: !item.is_visible } : item)
        : current.filter((item) => item.id !== rating.id));
      setMeta((current) => current && visibility !== "" ? { ...current, total_items: Math.max(0, current.total_items - 1) } : current);
      setFeedback(rating.is_visible ? "Ulasan disembunyikan dari publik." : "Ulasan ditampilkan kembali.");
    } catch (err) {
      setError(err.message || "Gagal mengubah status ulasan.");
    } finally {
      setUpdatingID(null);
    }
  };

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <Link to="/admin" className="mb-6 inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink">
        <Icon name="arrow_back" size={18} />
        Dashboard admin
      </Link>
      <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
        <div>
          <p className="mb-2 text-xs font-bold uppercase tracking-[0.12em] text-champagne">Moderasi konten</p>
          <h1 className="font-display text-3xl sm:text-4xl">Rating & ulasan</h1>
          <p className="mt-2 text-sm text-ink-muted">Sembunyikan ulasan yang melanggar kebijakan komunitas.</p>
        </div>
        <label className="text-sm">
          <span className="mb-1 block text-ink-muted">Tampilkan</span>
          <select
            value={visibility}
            onChange={(event) => {
              if (event.target.value === visibility) return;
              setError("");
              setFeedback("");
              setLoading(true);
              setPage(1);
              setVisibility(event.target.value);
            }}
            className="h-10 rounded-md border border-border bg-surface px-3"
          >
            <option value="">Semua ulasan</option>
            <option value="true">Ditampilkan</option>
            <option value="false">Disembunyikan</option>
          </select>
        </label>
      </div>

      {error && <p role="alert" className="mb-4 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700">{error}</p>}
      {feedback && <p role="status" className="mb-4 rounded-md bg-emerald-50 px-4 py-3 text-sm text-emerald-800">{feedback}</p>}

      {loading ? (
        <p className="py-16 text-center text-sm text-ink-muted">Memuat ulasan...</p>
      ) : ratings.length === 0 ? (
        <Card className="py-12 text-center text-sm text-ink-muted">Tidak ada ulasan pada filter ini.</Card>
      ) : (
        <div className="space-y-3">
          {ratings.map((rating) => (
            <Card key={rating.id}>
              <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-start">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <h2 className="font-semibold">{rating.field_name || `Lapangan #${rating.field_id}`}</h2>
                    <span className={`rounded-full px-2.5 py-1 text-xs ${rating.is_visible ? "bg-emerald-50 text-emerald-800" : "bg-canvas text-ink-muted"}`}>
                      {rating.is_visible ? "Ditampilkan" : "Disembunyikan"}
                    </span>
                  </div>
                  <p className="mt-1 text-xs text-ink-faint">
                    Booking #{rating.booking_id} · {formatDate(rating.created_at)}
                  </p>
                  <div className="mt-3 flex items-center gap-1" aria-label={`Rating ${rating.rating} dari 5`}>
                    {Array.from({ length: 5 }, (_, index) => (
                      <Icon key={index} name="star" size={17} filled={index < rating.rating} className={index < rating.rating ? "text-champagne" : "text-border"} />
                    ))}
                    <span className="ml-1 text-sm font-medium">{rating.rating}/5</span>
                  </div>
                  <p className="mt-3 whitespace-pre-line text-sm leading-6 text-ink-muted">
                    {rating.review || <span className="italic text-ink-faint">Tidak ada komentar.</span>}
                  </p>
                </div>
                <Button
                  variant={rating.is_visible ? "danger" : "secondary"}
                  size="sm"
                  disabled={updatingID === rating.id}
                  onClick={() => moderate(rating)}
                >
                  {updatingID === rating.id ? "Menyimpan..." : rating.is_visible ? "Sembunyikan" : "Tampilkan"}
                </Button>
              </div>
            </Card>
          ))}
        </div>
      )}

      {meta?.total_pages > 1 && (
        <div className="mt-5 flex items-center justify-between">
          <Button variant="secondary" size="sm" disabled={page <= 1 || loading} onClick={() => { setError(""); setFeedback(""); setLoading(true); setPage((value) => value - 1); }}>
            Sebelumnya
          </Button>
          <span className="text-sm text-ink-muted">Halaman {page} dari {meta.total_pages}</span>
          <Button variant="secondary" size="sm" disabled={page >= meta.total_pages || loading} onClick={() => { setError(""); setFeedback(""); setLoading(true); setPage((value) => value + 1); }}>
            Berikutnya
          </Button>
        </div>
      )}
    </div>
  );
}

function formatDate(value) {
  return new Intl.DateTimeFormat("id-ID", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}
