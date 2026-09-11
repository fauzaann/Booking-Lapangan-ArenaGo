import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams, Link } from "react-router-dom";
import Icon from "../components/ui/Icon";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import TimeSlotGrid from "../components/booking/TimeSlotGrid";
import { upcomingDays } from "../lib/mockData";
import { api } from "../lib/api";
import { formatDateInput, formatDateShort, formatIDR } from "../lib/format";

export default function SelectSlot() {
  const { courtId } = useParams();
  const navigate = useNavigate();
  const [court, setCourt] = useState(null);
  const [availability, setAvailability] = useState(null);
  const [error, setError] = useState("");

  const [dayIndex, setDayIndex] = useState(0);
  const [selected, setSelected] = useState([]);
  const selectedDate = upcomingDays[dayIndex];
  const dateValue = formatDateInput(selectedDate);

  useEffect(() => {
    setError("");
    Promise.all([api.field(courtId), api.availability(courtId, dateValue)])
      .then(([fieldResult, availabilityResult]) => {
        setCourt(fieldResult.data);
        setAvailability(availabilityResult.data);
      })
      .catch((err) => setError(err.message));
  }, [courtId, dateValue]);

  const slots = useMemo(
    () => (availability?.slots || []).map((slot) => ({
      time: slot.start_time,
      end: slot.end_time,
      status: slot.available ? "available" : "booked",
      vip: false,
    })),
    [availability]
  );

  function toggleSlot(time) {
    setSelected((prev) =>
      prev.includes(time) ? prev.filter((t) => t !== time) : [...prev, time].sort()
    );
  }

  const total = selected.length * (court?.pricePerHour || 0);

  function goToConfirmation() {
    navigate("/konfirmasi", {
      state: { court, date: selectedDate, slots: selected },
    });
  }

  if (error) return <div className="px-5 py-20 text-center text-sm text-red-600">{error}</div>;
  if (!court) return <div className="px-5 py-20 text-center text-sm text-ink-muted">Memuat lapangan...</div>;

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <Link
        to="/"
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink"
      >
        <Icon name="arrow_back" size={18} />
        Kembali ke Eksplor
      </Link>

      <div className="grid grid-cols-1 gap-10 lg:grid-cols-[1fr_360px]">
        <div>
          <p className="mb-1 text-xs font-bold uppercase tracking-[0.12em] text-champagne">
            {court.type}
          </p>
          <h1 className="mb-2 font-display text-3xl sm:text-4xl">{court.name}</h1>
          <p className="mb-8 text-ink-muted">{court.location} · {court.surface}</p>

          <h2 className="mb-3 text-sm font-semibold">Pilih Tanggal</h2>
          <div className="mb-8 flex gap-2.5 overflow-x-auto pb-2 no-scrollbar">
            {upcomingDays.map((d, i) => (
              <button
                key={i}
                onClick={() => {
                  setDayIndex(i);
                  setSelected([]);
                }}
                className={`flex min-w-[76px] flex-col items-center gap-1 rounded-md border px-3 py-3 transition-colors duration-150 ${
                  dayIndex === i
                    ? "border-onyx bg-onyx text-canvas"
                    : "border-border bg-surface-raised text-ink hover:border-sage"
                }`}
              >
                <span className="text-[11px] uppercase tracking-wide opacity-80">
                  {formatDateShort(d).split(",")[0]}
                </span>
                <span className="font-display text-lg leading-none">{d.getDate()}</span>
              </button>
            ))}
          </div>

          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-sm font-semibold">Pilih Jam</h2>
            <div className="flex items-center gap-3 text-xs text-ink-faint">
              <span className="flex items-center gap-1.5">
                <span className="h-1.5 w-1.5 rounded-full bg-champagne" /> Peak / VIP
              </span>
              <span className="flex items-center gap-1.5">
                <span className="h-1.5 w-1.5 rounded-full bg-slate" /> Terisi
              </span>
            </div>
          </div>
          {availability && !availability.is_open ? (
            <div className="rounded-lg border border-dashed border-border bg-surface-raised px-5 py-10 text-center">
              <Icon name="event_busy" size={28} className="mb-3 text-ink-faint" />
              <p className="mb-1 font-display text-lg">Lapangan tutup pada tanggal ini</p>
              <p className="text-sm text-ink-muted">Silakan pilih tanggal lain untuk melihat slot yang tersedia.</p>
            </div>
          ) : slots.length === 0 ? (
            <div className="rounded-lg border border-dashed border-border bg-surface-raised px-5 py-10 text-center text-sm text-ink-muted">
              Belum ada slot yang tersedia untuk tanggal ini.
            </div>
          ) : (
            <TimeSlotGrid slots={slots} selected={selected} onToggle={toggleSlot} />
          )}
        </div>

        <div className="lg:sticky lg:top-24 lg:self-start">
          <Card>
            <h3 className="mb-4 font-display text-xl">Ringkasan</h3>
            <div className="mb-4 space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-ink-muted">Tanggal</span>
                <span className="font-medium">{formatDateShort(upcomingDays[dayIndex])}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-ink-muted">Durasi</span>
                <span className="font-medium">{selected.length} jam</span>
              </div>
            </div>

            {selected.length > 0 ? (
              <div className="mb-4 flex flex-wrap gap-1.5">
                {selected.map((s) => (
                  <span
                    key={s}
                    className="rounded-full bg-sage-tint px-3 py-1 text-xs font-medium text-sage"
                  >
                    {s}
                  </span>
                ))}
              </div>
            ) : (
              <p className="mb-4 text-sm text-ink-faint">Belum ada jam dipilih.</p>
            )}

            <div className="mb-5 flex items-baseline justify-between border-t border-border pt-4">
              <span className="text-sm font-medium">Total</span>
              <span className="font-display text-2xl">{formatIDR(total)}</span>
            </div>

            <Button
              className="w-full"
              disabled={selected.length === 0}
              onClick={goToConfirmation}
            >
              Lanjut ke Konfirmasi
            </Button>
          </Card>
        </div>
      </div>
    </div>
  );
}
