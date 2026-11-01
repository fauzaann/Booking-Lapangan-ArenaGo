export const FIELD_DAYS = [
  { value: "MONDAY", label: "Senin" },
  { value: "TUESDAY", label: "Selasa" },
  { value: "WEDNESDAY", label: "Rabu" },
  { value: "THURSDAY", label: "Kamis" },
  { value: "FRIDAY", label: "Jumat" },
  { value: "SATURDAY", label: "Sabtu" },
  { value: "SUNDAY", label: "Minggu" },
];

export function createDefaultSchedules() {
  return FIELD_DAYS.map(({ value: day }) => ({
    day,
    open_time: "08:00",
    close_time: "22:00",
    is_closed: true,
  }));
}

export function normalizeSchedules(schedules = []) {
  const byDay = new Map(schedules.map((schedule) => [schedule.day, schedule]));
  return FIELD_DAYS.map(({ value: day }) => {
    const saved = byDay.get(day);
    return {
      day,
      open_time: saved?.open_time || "08:00",
      close_time: saved?.close_time || "22:00",
      is_closed: saved?.is_closed ?? true,
    };
  });
}

export function toFieldForm(field = {}, schedules = []) {
  return {
    name: field.name || "",
    description: field.description || "",
    location: field.location || "",
    price_per_hour: field.price_per_hour == null ? "" : String(field.price_per_hour),
    image_url: field.image_url || "",
    status: field.status || "ACTIVE",
    facilities: Array.isArray(field.facilities) ? field.facilities.join(", ") : "",
    schedules: normalizeSchedules(schedules),
  };
}

export function validateFieldForm(form) {
  const errors = {};
  const name = form.name.trim();
  const location = form.location.trim();
  const price = Number(form.price_per_hour);

  if (name.length < 3 || name.length > 120) errors.name = "Nama lapangan wajib 3–120 karakter.";
  if (location.length < 3 || location.length > 200) errors.location = "Lokasi wajib 3–200 karakter.";
  if (form.description.length > 1000) errors.description = "Deskripsi maksimal 1000 karakter.";
  if (!Number.isFinite(price) || price <= 0) errors.price_per_hour = "Harga per jam harus lebih besar dari 0.";

  const imageURL = form.image_url.trim();
  if (imageURL) {
    try {
      const url = new URL(imageURL);
      if (!["http:", "https:"].includes(url.protocol) || imageURL.length > 500) {
        errors.image_url = "Masukkan URL gambar HTTP/HTTPS maksimal 500 karakter.";
      }
    } catch {
      errors.image_url = "Masukkan URL gambar HTTP/HTTPS yang valid.";
    }
  }

  const facilities = form.facilities.split(",").map((item) => item.trim()).filter(Boolean);
  if (facilities.some((facility) => facility.length > 50)) {
    errors.facilities = "Setiap fasilitas maksimal 50 karakter.";
  }

  if (form.schedules.length !== FIELD_DAYS.length) {
    errors.schedules = "Atur jadwal untuk setiap hari.";
  } else {
    const seenDays = new Set();
    for (const schedule of form.schedules) {
      if (!FIELD_DAYS.some(({ value }) => value === schedule.day) || seenDays.has(schedule.day)) {
        errors.schedules = "Setiap hari harus memiliki satu jadwal.";
        break;
      }
      seenDays.add(schedule.day);
      if (!isValidTime(schedule.open_time) || !isValidTime(schedule.close_time)) {
        errors.schedules = "Jam operasional harus menggunakan format HH:MM yang valid.";
        break;
      }
      if (!schedule.is_closed && minutes(schedule.open_time) >= minutes(schedule.close_time)) {
        errors.schedules = "Jam buka harus lebih awal dari jam tutup.";
        break;
      }
      if (!schedule.is_closed && (minutes(schedule.close_time) - minutes(schedule.open_time)) % 60 !== 0) {
        errors.schedules = "Durasi jam operasional harus dalam kelipatan satu jam.";
        break;
      }
    }
  }

  return errors;
}

export function toFieldPayload(form) {
  return {
    name: form.name.trim(),
    description: form.description.trim(),
    type: "PADEL",
    location: form.location.trim(),
    price_per_hour: Number(form.price_per_hour),
    image_url: form.image_url.trim(),
    status: form.status,
    facilities: form.facilities.split(",").map((item) => item.trim()).filter(Boolean),
    schedules: form.schedules,
  };
}

function isValidTime(value) {
  if (!/^\d{2}:\d{2}$/.test(value)) return false;
  const [hour, minute] = value.split(":").map(Number);
  return hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59;
}

function minutes(value) {
  const [hour, minute] = value.split(":").map(Number);
  return hour * 60 + minute;
}
