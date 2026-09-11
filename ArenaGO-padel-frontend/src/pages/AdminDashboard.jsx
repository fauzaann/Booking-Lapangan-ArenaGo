import { useMemo, useState } from "react";
import Icon from "../components/ui/Icon";
import Card from "../components/ui/Card";
import Button from "../components/ui/Button";
import Modal from "../components/ui/Modal";
import { courts, buildDaySlots, upcomingDays, adminStats } from "../lib/mockData";
import { formatDateShort } from "../lib/format";

export default function AdminDashboard() {
  const [courtId, setCourtId] = useState(courts[0].id);
  const [dayIndex, setDayIndex] = useState(0);
  const [overrides, setOverrides] = useState({});
  const [activeSlot, setActiveSlot] = useState(null);

  const baseSlots = useMemo(() => buildDaySlots(), [courtId, dayIndex]);
  const key = (time) => `${courtId}-${dayIndex}-${time}`;

  const slots = baseSlots.map((s) => ({
    ...s,
    status: overrides[key(s.time)] ?? s.status,
  }));

  function toggleBlock(time, currentStatus) {
    const k = key(time);
    setOverrides((prev) => ({
      ...prev,
      [k]: currentStatus === "blocked" ? "available" : "blocked",
    }));
    setActiveSlot(null);
  }

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <h1 className="mb-2 font-display text-3xl sm:text-4xl">Kelola Jadwal Lapangan</h1>
      <p className="mb-8 text-ink-muted">
        Blokir slot untuk maintenance, event privat, atau ketidaktersediaan lapangan.
      </p>

      <div className="mb-8 grid grid-cols-2 gap-4 lg:grid-cols-4">
        {adminStats.map((stat) => (
          <Card key={stat.label} padded className="p-5">
            <p className="mb-1 text-xs uppercase tracking-[0.1em] text-ink-faint">
              {stat.label}
            </p>
            <p className="mb-1 font-display text-2xl">{stat.value}</p>
            <p className="text-xs text-sage">{stat.delta}</p>
          </Card>
        ))}
      </div>

      <div className="mb-6 flex flex-wrap gap-2.5">
        {courts.map((c) => (
          <button
            key={c.id}
            onClick={() => setCourtId(c.id)}
            className={`rounded-full border px-4 py-2 text-sm font-medium transition-colors duration-150 ${
              courtId === c.id
                ? "border-onyx bg-onyx text-canvas"
                : "border-border bg-surface-raised text-ink-muted hover:text-ink"
            }`}
          >
            {c.name}
          </button>
        ))}
      </div>

      <div className="mb-6 flex gap-2.5 overflow-x-auto pb-2 no-scrollbar">
        {upcomingDays.map((d, i) => (
          <button
            key={i}
            onClick={() => setDayIndex(i)}
            className={`flex min-w-[76px] flex-col items-center gap-1 rounded-md border px-3 py-3 ${
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

      <Card>
        <div className="mb-5 flex items-center justify-between">
          <h2 className="font-display text-xl">Grid Slot — {formatDateShort(upcomingDays[dayIndex])}</h2>
          <div className="flex items-center gap-4 text-xs text-ink-faint">
            <Legend color="bg-surface-raised border border-border" label="Tersedia" />
            <Legend color="bg-slate/40" label="Terisi" />
            <Legend color="bg-terracotta/80" label="Diblokir" />
          </div>
        </div>

        <div className="grid grid-cols-3 gap-2.5 sm:grid-cols-4 lg:grid-cols-7">
          {slots.map((slot) => (
            <button
              key={slot.time}
              onClick={() => slot.status !== "booked" && setActiveSlot(slot)}
              disabled={slot.status === "booked"}
              className={`flex h-16 flex-col items-center justify-center rounded-md border text-sm font-medium transition-colors duration-150 ${
                slot.status === "booked"
                  ? "cursor-not-allowed border-transparent bg-slate/20 text-ink-faint"
                  : slot.status === "blocked"
                  ? "border-terracotta/40 bg-terracotta/10 text-terracotta"
                  : "border-border bg-surface-raised text-ink hover:border-onyx"
              }`}
            >
              <span className="flex items-center gap-1">
                {slot.status === "blocked" && <Icon name="lock" size={13} />}
                {slot.time}
              </span>
              <span className="mt-0.5 text-[10px] font-normal uppercase tracking-wide opacity-70">
                {slot.status === "booked" ? "Terisi" : slot.status === "blocked" ? "Blokir" : "Tersedia"}
              </span>
            </button>
          ))}
        </div>
      </Card>

      <Modal
        open={!!activeSlot}
        onClose={() => setActiveSlot(null)}
        title={activeSlot ? `Slot ${activeSlot.time}` : ""}
        footer={
          activeSlot && (
            <div className="flex gap-3">
              <Button variant="secondary" className="flex-1" onClick={() => setActiveSlot(null)}>
                Batal
              </Button>
              <Button
                variant={activeSlot.status === "blocked" ? "primary" : "danger"}
                className="flex-1"
                onClick={() => toggleBlock(activeSlot.time, activeSlot.status)}
              >
                {activeSlot.status === "blocked" ? "Buka Kembali Slot" : "Blokir Slot"}
              </Button>
            </div>
          )
        }
      >
        {activeSlot && (
          <p className="text-sm text-ink-muted">
            {activeSlot.status === "blocked"
              ? "Slot ini sedang diblokir dan tidak dapat dipesan pelanggan. Buka kembali agar tersedia untuk booking."
              : "Blokir slot ini untuk maintenance, latihan internal, atau event privat. Pelanggan tidak akan bisa memesan slot yang diblokir."}
          </p>
        )}
      </Modal>
    </div>
  );
}

function Legend({ color, label }) {
  return (
    <span className="flex items-center gap-1.5">
      <span className={`h-3 w-3 rounded-[3px] ${color}`} />
      {label}
    </span>
  );
}
