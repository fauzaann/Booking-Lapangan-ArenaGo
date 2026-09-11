import { formatIDR, formatDateLong } from "../../lib/format";
import { bookingPriceBreakdown } from "../../lib/mockData";

export default function BookingSummary({ booking, showBreakdown = true }) {
  const breakdown = bookingPriceBreakdown(booking);
  const total = breakdown.find((b) => b.label === "__total__")?.amount;
  const lines = breakdown.filter((b) => b.label !== "__total__");

  return (
    <div>
      <div className="mb-4">
        <p className="text-xs uppercase tracking-[0.12em] text-ink-faint">Lapangan</p>
        <p className="font-display text-lg">{booking.court.name}</p>
        <p className="text-sm text-ink-muted">{booking.court.location}</p>
      </div>

      <div className="mb-4 grid grid-cols-2 gap-4 border-t border-border pt-4">
        <div>
          <p className="text-xs uppercase tracking-[0.12em] text-ink-faint">Tanggal</p>
          <p className="text-sm font-medium">{formatDateLong(booking.date)}</p>
        </div>
        <div>
          <p className="text-xs uppercase tracking-[0.12em] text-ink-faint">Jam</p>
          <p className="text-sm font-medium">
            {booking.slots[0]} – {addHour(booking.slots[booking.slots.length - 1])}
          </p>
        </div>
      </div>

      {showBreakdown && (
        <div className="space-y-2.5 border-t border-border pt-4">
          {lines.map((line) => (
            <div key={line.label} className="flex justify-between text-sm text-ink-muted">
              <span>{line.label}</span>
              <span>{formatIDR(line.amount)}</span>
            </div>
          ))}
        </div>
      )}

      <div className="mt-4 flex items-baseline justify-between border-t border-border pt-4">
        <span className="text-sm font-medium">Total Bayar</span>
        <span className="font-display text-2xl">{formatIDR(total)}</span>
      </div>
    </div>
  );
}

function addHour(time) {
  const [h, m] = time.split(":").map(Number);
  return `${String((h + 1) % 24).padStart(2, "0")}:${String(m).padStart(2, "0")}`;
}
