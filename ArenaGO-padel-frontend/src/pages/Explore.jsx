import { useEffect, useMemo, useState } from "react";
import Icon from "../components/ui/Icon";
import Tabs from "../components/ui/Tabs";
import CourtCard from "../components/booking/CourtCard";
import { courts } from "../lib/mockData";

const LOCATIONS = [
  { value: "all", label: "Semua Kota" },
  { value: "jakarta", label: "Jakarta" },
  { value: "bali", label: "Bali" },
];

export default function Explore() {
  const [location, setLocation] = useState("all");
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(true);

  // Simulate a brief network fetch whenever the location filter changes,
  // including the very first load. Pure UI state — no real request.
  useEffect(() => {
    setLoading(true);
    const timer = setTimeout(() => setLoading(false), 500);
    return () => clearTimeout(timer);
  }, [location]);

  const filtered = useMemo(() => {
    return courts.filter((c) => {
      const matchLocation =
        location === "all" || c.location.toLowerCase().includes(location);
      const matchQuery =
        !query || c.name.toLowerCase().includes(query.toLowerCase());
      return matchLocation && matchQuery;
    });
  }, [location, query]);

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <section className="mb-10">
        <p className="mb-3 text-xs font-bold uppercase tracking-[0.12em] text-champagne">
          Jakarta & Bali
        </p>
        <h1 className="mb-3 font-display text-3xl leading-tight sm:text-5xl">
          Reservasi lapangan padel pilihan, tanpa ribet.
        </h1>
        <p className="max-w-xl text-ink-muted">
          Pilih lapangan, atur jadwal, dan bayar dalam hitungan menit. Setiap
          court dikurasi untuk pengalaman bermain yang tenang dan premium.
        </p>
      </section>

      <section className="mb-8 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative w-full sm:max-w-sm">
          <Icon
            name="search"
            className="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-ink-faint"
          />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Cari nama lapangan..."
            className="h-12 w-full rounded-full border border-border bg-surface-raised pl-11 pr-4 text-sm focus:border-onyx focus:outline-none"
          />
        </div>
        <Tabs
          tabs={LOCATIONS}
          active={location}
          onChange={setLocation}
          className="self-start sm:self-auto"
        />
      </section>

      {loading ? (
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
    </div>
  );
}
