export function toggleContiguousSlot(slots, selected, time) {
  const index = slots.findIndex((slot) => slot.time === time);
  if (index < 0 || slots[index].status !== "available") return selected;
  if (selected.length === 0) return [time];

  const selectedIndexes = selected
    .map((selectedTime) => slots.findIndex((slot) => slot.time === selectedTime))
    .filter((selectedIndex) => selectedIndex >= 0);
  if (selectedIndexes.length === 0) return [time];

  const firstIndex = Math.min(...selectedIndexes);
  const lastIndex = Math.max(...selectedIndexes);

  if (selected.includes(time)) {
    if (selected.length === 1) return [];
    if (index === firstIndex || index === lastIndex) {
      return selected.filter((selectedTime) => selectedTime !== time);
    }
    return [time];
  }

  if (index === firstIndex - 1) return [time, ...selected];
  if (index === lastIndex + 1) return [...selected, time];
  return [time];
}
