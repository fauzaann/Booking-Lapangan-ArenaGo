import { Link } from "react-router-dom";
import Icon from "../ui/Icon";
import { formatIDR } from "../../lib/format";

export default function CourtCard({ court }) {
  return (
    <Link
      to={`/jadwal/${court.id}`}
      className="group block overflow-hidden rounded-lg border border-border bg-surface transition-shadow duration-200 hover:shadow-raised"
    >
      <div
        className={`relative aspect-[16/10] w-full overflow-hidden bg-gradient-to-br ${court.gradient}`}
      >
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_30%_20%,rgba(255,255,255,0.12),transparent_60%)]" />
        <div className="absolute inset-0 flex items-end p-5">
          <span className="font-display text-2xl text-canvas">{court.name.split("—")[0]}</span>
        </div>
        {court.vip && (
          <span className="absolute right-4 top-4 inline-flex items-center gap-1 rounded-full bg-champagne px-3 py-1 text-[11px] font-bold uppercase tracking-[0.12em] text-onyx">
            <Icon name="workspace_premium" size={14} />
            VIP
          </span>
        )}
        <div className="absolute left-4 top-4 flex items-center gap-1 rounded-full bg-canvas/90 px-2.5 py-1 text-xs font-semibold">
          <Icon name="star" size={14} filled className="text-champagne" />
          {court.rating}
        </div>
      </div>

      <div className="p-6">
        <div className="mb-1 flex items-center gap-1.5 text-xs text-ink-faint">
          <Icon name="location_on" size={14} />
          {court.location}
        </div>
        <h3 className="mb-2 font-display text-xl">{court.name}</h3>
        <div className="mb-4 flex flex-wrap gap-1.5">
          {court.tags.map((tag) => (
            <span
              key={tag}
              className="rounded-full border border-border px-2.5 py-1 text-[11px] text-ink-muted"
            >
              {tag}
            </span>
          ))}
        </div>
        <div className="flex items-end justify-between border-t border-border pt-4">
          <div className="text-xs text-ink-faint">
            {court.type} · {court.surface}
          </div>
          <div className="text-right">
            <div className="font-display text-lg leading-none">{formatIDR(court.pricePerHour)}</div>
            <div className="text-[11px] text-ink-faint">/ jam</div>
          </div>
        </div>
      </div>
    </Link>
  );
}
