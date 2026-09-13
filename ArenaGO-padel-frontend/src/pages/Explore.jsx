import { useEffect, useMemo, useState } from "react";
import Icon from "../components/ui/Icon";
import Tabs from "../components/ui/Tabs";
import CourtCard from "../components/booking/CourtCard";
import { api } from "../lib/api";

const LOCATIONS = [
  { value: "all", label: "Semua Kota" },
  { value: "jakarta", label: "Jakarta" },
  { value: "bali", label: "Bali" },
];

export default function Explore() {
  const [location, setLocation] = useState("all");
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);

  const [courts, setCourts] = useState([]);
  const [error, setError] = useState("");

  useEffect(() => {
    setLoading(true);
    setError("");
    api.fields(location === "all" ? {} : { location })
      .then((result) => setCourts(result.data))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [location]);

  const filtered = useMemo(() => {
    return courts.filter((c) => {
      const matchLocation =
        location === "all" || c.location.toLowerCase().includes(location);
      const matchQuery =
        !query || c.name.toLowerCase().includes(query.toLowerCase());
      return matchLocation && matchQuery;
    });
  }, [courts, location, query]);

  return (
    <div className="pb-12">
      <section
        className="relative isolate overflow-hidden bg-sage text-canvas"
        style={{ backgroundImage: "url(https://images.unsplash.com/photo-1554068865-24cecd4e34b8?auto=format&fit=crop&w=2200&q=85)" }}
      >
        <div className="absolute inset-0 -z-10 bg-[linear-gradient(90deg,rgba(20,23,26,0.92),rgba(45,75,62,0.66),rgba(45,75,62,0.2))]" />
        <div className="absolute inset-0 -z-10 bg-gradient-to-t from-sage/80 via-transparent to-onyx/20" />
        <div className="mx-auto flex min-h-[480px] max-w-[1440px] items-end px-5 py-12 sm:min-h-[540px] sm:px-8 sm:py-16 lg:px-16 lg:py-20">
          <div className="max-w-3xl">
            <p className="mb-4 text-xs font-bold uppercase tracking-[0.18em] text-champagne">
              ArenaGO Padel · Jakarta & Bali
            </p>
            <h1 className="max-w-2xl font-display text-4xl leading-[1.05] sm:text-6xl lg:text-7xl">
              Temukan court untuk permainan terbaikmu.
            </h1>
            <p className="mt-5 max-w-xl text-sm leading-7 text-canvas/78 sm:text-base">
              Pilih lapangan yang pas, amankan jadwalnya, lalu datang dan main tanpa drama.
              Setiap sesi dimulai dari satu booking yang sederhana.
            </p>
            <div className="mt-8 flex flex-wrap gap-3 text-xs font-semibold uppercase tracking-[0.1em] text-canvas/75">
              <span className="rounded-full border border-canvas/30 bg-canvas/10 px-3 py-2">Lapangan terkurasi</span>
              <span className="rounded-full border border-canvas/30 bg-canvas/10 px-3 py-2">Jadwal real-time</span>
              <span className="rounded-full border border-canvas/30 bg-canvas/10 px-3 py-2">Pembayaran aman</span>
            </div>
          </div>
        </div>
      </section>

      <section className="relative z-10 mx-auto -mt-7 mb-12 flex max-w-[1440px] flex-col gap-4 rounded-lg border border-border bg-surface-raised p-4 shadow-overlay sm:-mt-8 sm:flex-row sm:items-center sm:justify-between sm:px-5 lg:mx-16 xl:mx-auto">
        <div className="relative w-full sm:max-w-md">
          <Icon
            name="search"
            className="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-ink-faint"
          />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Cari lapangan atau lokasi..."
            className="h-12 w-full rounded-full border border-border bg-surface pl-11 pr-4 text-sm focus:border-onyx focus:outline-none"
          />
        </div>
        <Tabs
          tabs={LOCATIONS}
          active={location}
          onChange={setLocation}
          className="self-start sm:self-auto"
        />
      </section>

      <section className="mx-auto max-w-[1440px] px-5 sm:px-8 lg:px-16">
        <div className="mb-6 flex items-end justify-between gap-4">
          <div>
            <p className="mb-2 text-xs font-bold uppercase tracking-[0.14em] text-champagne">Pilihan untukmu</p>
            <h2 className="font-display text-3xl sm:text-4xl">Lapangan yang sedang siap dimainkan</h2>
          </div>
          {!loading && !error && <p className="hidden text-sm text-ink-muted sm:block">{filtered.length} lapangan tersedia</p>}
        </div>

        {error ? (
          <div className="rounded-lg border border-dashed border-border py-20 text-center text-sm text-red-600">{error}</div>
      ) : loading ? (
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <div
              key={i}
              className="h-[380px] animate-pulse rounded-lg border border-border bg-surface"
            />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-20 text-center">
          <Icon name="search_off" size={32} className="mb-3 text-ink-faint" />
          <p className="mb-1 font-display text-xl">Lapangan tidak ditemukan</p>
          <p className="text-sm text-ink-muted">
            Coba ubah kata kunci atau pilih kota lain.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 xl:grid-cols-3">
          {filtered.map((court) => (
            <CourtCard key={court.id} court={court} />
          ))}
        </div>
      )}
      </section>
    </div>
  );
}
