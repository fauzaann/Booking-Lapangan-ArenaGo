const WEEKDAYS = {
  minggu: 0,
  ahad: 0,
  senin: 1,
  selasa: 2,
  rabu: 3,
  kamis: 4,
  jumat: 5,
  sabtu: 6,
};

const MONTHS = {
  januari: 0,
  februari: 1,
  maret: 2,
  april: 3,
  mei: 4,
  juni: 5,
  juli: 6,
  agustus: 7,
  september: 8,
  oktober: 9,
  november: 10,
  desember: 11,
};

const AVAILABILITY_TERMS = /\b(kosong|tersedia|availability|available|slot)\b/i;
const DATE_REFERENCE_TERMS = /\b(tanggal|hari)\s+(itu|tersebut|tadi)\b/i;
const TIME_REFERENCE_TERMS = /\b(jam|waktu)\s+(itu|tersebut|tadi)\b/i;
const CLOCK_PATTERN = String.raw`(\d{1,2})(?::(\d{2})|[.](\d{2}))?(?:\s*(pagi|siang|sore|malam))?`;

export function isAvailabilityQuestion(message) {
  return AVAILABILITY_TERMS.test(message);
}

export function parseAvailabilityRequest(messages, now = new Date()) {
  const userMessages = messages
    .filter((message) => message.from === "user")
    .map((message) => message.text);
  const currentMessage = userMessages.at(-1) || "";

  let date = parseDate(currentMessage, now);
  if (!date && (DATE_REFERENCE_TERMS.test(currentMessage) || TIME_REFERENCE_TERMS.test(currentMessage))) {
    for (let index = userMessages.length - 2; index >= 0; index -= 1) {
      date = parseDate(userMessages[index], now);
      if (date) break;
    }
  }

  let slots = parseRequestedSlots(currentMessage);
  if (!slots && TIME_REFERENCE_TERMS.test(currentMessage)) {
    for (let index = userMessages.length - 2; index >= 0; index -= 1) {
      slots = parseRequestedSlots(userMessages[index]);
      if (slots) break;
    }
  }

  return {
    date: date ? formatDateInput(date) : null,
    slots,
    needsTimeClarification: TIME_REFERENCE_TERMS.test(currentMessage) && !slots,
  };
}

function parseDate(message, now) {
  const normalized = message.toLowerCase();
  const base = new Date(now.getFullYear(), now.getMonth(), now.getDate());

  if (/\b(lusa)\b/.test(normalized)) {
    base.setDate(base.getDate() + 2);
    return base;
  }
  if (/\b(besok)\b/.test(normalized)) {
    base.setDate(base.getDate() + 1);
    return base;
  }
  if (/\b(hari ini|sekarang)\b/.test(normalized)) return base;

  const iso = normalized.match(/\b(\d{4})-(\d{2})-(\d{2})\b/);
  if (iso) return validDate(Number(iso[1]), Number(iso[2]), Number(iso[3]));

  const numeric = normalized.match(/\b(\d{1,2})[/-](\d{1,2})[/-](\d{4})\b/);
  if (numeric) return validDate(Number(numeric[3]), Number(numeric[2]), Number(numeric[1]));

  const textual = normalized.match(
    /\b(?:tanggal\s+)?(\d{1,2})\s+(januari|februari|maret|april|mei|juni|juli|agustus|september|oktober|november|desember)(?:\s+(\d{4}))?\b/,
  );
  if (textual) {
    const day = Number(textual[1]);
    const month = MONTHS[textual[2]];
    const year = textual[3] ? Number(textual[3]) : base.getFullYear();
    const result = validDate(year, month + 1, day);
    if (result && !textual[3] && result < base) result.setFullYear(result.getFullYear() + 1);
    return result;
  }

  for (const [weekday, dayNumber] of Object.entries(WEEKDAYS)) {
    if (!new RegExp(`\\b${weekday}\\b`).test(normalized)) continue;
    const daysAhead = (dayNumber - base.getDay() + 7) % 7 || 7;
    base.setDate(base.getDate() + daysAhead);
    return base;
  }
  return null;
}

function validDate(year, month, day) {
  const value = new Date(year, month - 1, day);
  if (value.getFullYear() !== year || value.getMonth() !== month - 1 || value.getDate() !== day) return null;
  return value;
}

function parseRequestedSlots(message) {
  const normalized = message.toLowerCase();
  const startMatch = normalized.match(new RegExp(`(?:\\b(?:jam|pukul)\\s*)${CLOCK_PATTERN}`, "i"))
    || normalized.match(/\b(\d{1,2})(?::(\d{2})|[.](\d{2}))(?:\s*(pagi|siang|sore|malam))?\b/i);
  if (!startMatch) return null;

  const startClock = parseClock(startMatch);
  if (!startClock || startClock.hour >= 24) return null;

  const afterStart = normalized.slice(startMatch.index + startMatch[0].length);
  const endMatch = afterStart.match(new RegExp(`^\\s*(?:sampai|hingga|s/d|[-–])\\s*(?:jam\\s*)?${CLOCK_PATTERN}`, "i"));
  let startHour = startClock.hour;
  let endHour = startClock.hour + 1;
  if (endMatch) {
    const endClock = parseClock(endMatch);
    if (!endClock) return null;
    endHour = endClock.hour;
    if (!startClock.period && endClock.period && endClock.hour >= 12 && startHour < 12) {
      startHour += 12;
    }
    if (!endClock.period && startClock.period && endClock.hour < 12 && startClock.hour >= 12) {
      endHour += 12;
    }
  }
  if (endHour <= startHour || endHour > 24) return null;

  return Array.from({ length: endHour - startHour }, (_, index) => `${String(startHour + index).padStart(2, "0")}:00`);
}

function parseClock(match) {
  const hour = Number(match[1]);
  const minute = Number(match[2] || match[3] || 0);
  const period = match[4];
  if (minute !== 0 || hour > 24 || (hour === 24 && period)) return null;

  let normalizedHour = hour;
  if (period === "malam" || period === "sore") {
    if (normalizedHour < 12) normalizedHour += 12;
  } else if (period === "pagi" && normalizedHour === 12) {
    normalizedHour = 0;
  } else if (period === "siang" && normalizedHour < 11) {
    normalizedHour += 12;
  }
  return { hour: normalizedHour, period };
}

function formatDateInput(date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}
