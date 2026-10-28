import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import Icon from "../components/ui/Icon";
import { api } from "../lib/api";
import { formatIDR } from "../lib/format";

const PAGE_SIZE = 10;

export default function FieldDetail() {
  const { id } = useParams();
  const [field, setField] = useState(null);
  const [summary, setSummary] = useState(null);
  const [reviews, setReviews] = useState([]);
  const [meta, setMeta] = useState(null);
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    Promise.all([
      api.field(id),
      api.fieldRatingSummary(id),
      api.fieldRatings(id, { page: String(page), limit: String(PAGE_SIZE) }),
    ])
      .then(([fieldResult, summaryResult, reviewsResult]) => {
        if (!active) return;
        setError("");
        setField(fieldResult.data);
        setSummary(summaryResult.data);
        setReviews(reviewsResult.data);
        setMeta(reviewsResult.meta);
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
  }, [id, page]);

  if (loading || (field && String(field.id) !== String(id))) return <p className="px-5 py-20 text-center text-sm text-ink-muted">Memuat detail lapangan...</p>;
  if (error) return <p role="alert" className="px-5 py-20 text-center text-sm text-red-600">{error}</p>;
  if (!field) return null;

  const hasRating = summary?.total_reviews > 0;
  const totalPages = meta?.total_pages || 0;

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <Link to="/" className="mb-6 inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink">
        <Icon name="arrow_back" size={18} />
        Kembali ke lapangan
      </Link>

      <div className="grid gap-8 lg:grid-cols-[1.2fr_0.8fr]">
        <div>
          <div className="relative mb-6 aspect-[16/9] overflow-hidden rounded-lg bg-sage">
            {field.image_url ? (
              <img src={field.image_url} alt={field.name} className="h-full w-full object-cover" />
            ) : (
              <div className="flex h-full items-center justify-center text-canvas">
                <Icon name="sports_tennis" size={64} />
              </div>
            )}
          </div>
          <p className="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-champagne">{field.location}</p>
          <h1 className="font-display text-3xl sm:text-4xl">{field.name}</h1>
          <p className="mt-4 whitespace-pre-line text-sm leading-7 text-ink-muted">
            {field.description || "Lapangan untuk sesi bermain yang nyaman."}
          </p>
          <div className="mt-5 flex flex-wrap gap-2">
            {(field.facilities || []).map((facility) => (
              <span key={facility} className="rounded-full border border-border px-3 py-1.5 text-xs text-ink-muted">
                {facility}
              </span>
            ))}
          </div>
        </div>

        <div className="space-y-5">
          <Card>
            <p className="text-sm text-ink-muted">Harga mulai</p>
            <p className="mt-1 font-display text-2xl">{formatIDR(field.price_per_hour)} <span className="text-sm font-normal text-ink-faint">/ jam</span></p>
            {hasRating ? (
              <div className="mt-4 flex items-center gap-2 text-sm">
                <Icon name="star" filled className="text-champagne" />
                <span className="font-semibold">{Number(summary.average_rating).toFixed(1)}</span>
                <span className="text-ink-muted">· {summary.total_reviews} ulasan</span>
              </div>
            ) : (
              <p className="mt-4 text-sm text-ink-faint">Belum ada ulasan.</p>
            )}
            {field.status === "ACTIVE" ? (
              <Link to={`/jadwal/${field.id}`} className="mt-5 block">
                <Button className="w-full">Pilih Jadwal</Button>
              </Link>
            ) : (
              <p role="status" className="mt-5 rounded-md bg-canvas px-3 py-2 text-sm text-ink-muted">
                Lapangan sedang tidak menerima booking.
              </p>
            )}
          </Card>

          {field.schedules?.length > 0 && (
            <Card>
              <h2 className="mb-4 font-display text-xl">Jam operasional</h2>
              <ul className="space-y-2 text-sm">
                {field.schedules.map((schedule) => (
                  <li key={schedule.id} className="flex justify-between gap-4">
                    <span className="text-ink-muted">{schedule.day}</span>
                    <span>{schedule.is_closed ? "Tutup" : `${schedule.open_time}–${schedule.close_time}`}</span>
                  </li>
                ))}
              </ul>
            </Card>
          )}
        </div>
      </div>

      <section className="mt-12">
        <div className="mb-5 flex flex-wrap items-end justify-between gap-3">
          <div>
            <p className="mb-1 text-xs font-bold uppercase tracking-[0.14em] text-champagne">Pengalaman pelanggan</p>
            <h2 className="font-display text-2xl sm:text-3xl">Rating & ulasan</h2>
          </div>
          {hasRating && <p className="text-sm text-ink-muted">{summary.total_reviews} ulasan terverifikasi</p>}
        </div>

        {reviews.length === 0 ? (
          <Card className="text-center text-sm text-ink-muted">Belum ada ulasan untuk lapangan ini.</Card>
        ) : (
          <div className="space-y-3">
            {reviews.map((review) => (
              <Card key={review.id}>
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div>
                    <p className="font-semibold">{review.author}</p>
                    <p className="mt-1 text-xs text-ink-faint">{formatReviewDate(review.created_at)}</p>
                  </div>
                  <div className="flex items-center gap-1" aria-label={`Rating ${review.rating} dari 5`}>
                    {Array.from({ length: 5 }, (_, index) => (
                      <Icon key={index} name="star" filled={index < review.rating} size={18} className={index < review.rating ? "text-champagne" : "text-border"} />
                    ))}
                  </div>
                </div>
                {review.review && <p className="mt-4 whitespace-pre-line text-sm leading-6 text-ink-muted">{review.review}</p>}
              </Card>
            ))}
          </div>
        )}

        {totalPages > 1 && (
          <div className="mt-5 flex items-center justify-between">
            <Button variant="secondary" size="sm" disabled={page <= 1} onClick={() => { setError(""); setLoading(true); setPage((value) => value - 1); }}>
              Sebelumnya
            </Button>
            <span className="text-sm text-ink-muted">Halaman {page} dari {totalPages}</span>
            <Button variant="secondary" size="sm" disabled={page >= totalPages} onClick={() => { setError(""); setLoading(true); setPage((value) => value + 1); }}>
              Berikutnya
            </Button>
          </div>
        )}
      </section>
    </div>
  );
}

function formatReviewDate(value) {
  return new Intl.DateTimeFormat("id-ID", { dateStyle: "medium" }).format(new Date(value));
}
