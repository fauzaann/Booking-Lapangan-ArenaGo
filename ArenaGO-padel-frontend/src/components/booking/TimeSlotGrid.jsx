import Icon from "../ui/Icon";

function SlotChip({ slot, selected, onToggle }) {
  const isBooked = slot.status === "booked";

  const base =
    "relative flex h-12 min-w-[84px] items-center justify-center gap-1.5 rounded-full border px-4 text-sm font-medium transition-all duration-150";

  let style;
  if (isBooked) {
    style = "cursor-not-allowed border-transparent bg-surface text-ink-faint line-through";
  } else if (selected) {
    style = "border-sage bg-sage text-canvas shadow-[0_4px_12px_rgba(45,75,62,0.2)]";
  } else {
    style = "border-border bg-surface-raised text-ink hover:border-sage";
  }

  return (
    <button
      type="button"
      disabled={isBooked}
      onClick={() => onToggle(slot.time)}
      className={`${base} ${style}`}
    >
      {slot.vip && !isBooked && (
        <span className="h-1.5 w-1.5 rounded-full bg-champagne" aria-hidden="true" />
      )}
      {slot.time}
      {selected && <Icon name="check" size={16} />}
    </button>
  );
}

export default function TimeSlotGrid({ slots, selected, onToggle }) {
  return (
    <div className="flex flex-wrap gap-2.5">
      {slots.map((slot) => (
        <SlotChip
          key={slot.time}
          slot={slot}
          selected={selected.includes(slot.time)}
          onToggle={onToggle}
        />
      ))}
    </div>
  );
}
