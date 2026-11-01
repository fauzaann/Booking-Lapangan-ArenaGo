import test from "node:test";
import assert from "node:assert/strict";

import {
  createDefaultSchedules,
  normalizeSchedules,
  toFieldForm,
  toFieldPayload,
  validateFieldForm,
} from "./adminFields.js";

function validForm() {
  return {
    name: "Lapangan Padel A",
    description: "Lapangan indoor",
    location: "Jakarta Selatan",
    price_per_hour: "200000",
    image_url: "https://example.com/court.jpg",
    status: "ACTIVE",
    facilities: "Shower, Parkir",
    schedules: createDefaultSchedules(),
  };
}

test("new fields default to closed schedules for every weekday", () => {
  const schedules = createDefaultSchedules();
  assert.equal(schedules.length, 7);
  assert.ok(schedules.every((schedule) => schedule.is_closed));
});

test("normalizes saved schedules and defaults missing weekdays to closed", () => {
  const schedules = normalizeSchedules([{ day: "MONDAY", open_time: "09:00", close_time: "21:00", is_closed: false }]);
  assert.equal(schedules[0].open_time, "09:00");
  assert.equal(schedules[0].is_closed, false);
  assert.ok(schedules.slice(1).every((schedule) => schedule.is_closed));
});

test("converts field form values to backend payload", () => {
  const payload = toFieldPayload(validForm());
  assert.equal(payload.price_per_hour, 200000);
  assert.deepEqual(payload.facilities, ["Shower", "Parkir"]);
  assert.equal(payload.schedules.length, 7);
});

test("accepts a complete valid field form", () => {
  const form = validForm();
  form.schedules[0] = { ...form.schedules[0], is_closed: false };
  assert.deepEqual(validateFieldForm(form), {});
});

test("rejects invalid price, image URL, and contradictory hours", () => {
  const form = validForm();
  form.price_per_hour = "0";
  form.image_url = "javascript:alert(1)";
  form.schedules[0] = { ...form.schedules[0], is_closed: false, open_time: "22:00", close_time: "08:00" };

  const errors = validateFieldForm(form);
  assert.ok(errors.price_per_hour);
  assert.ok(errors.image_url);
  assert.ok(errors.schedules);
});

test("rejects operating hours that are not a whole number of hours", () => {
  const form = validForm();
  form.schedules[0] = { ...form.schedules[0], is_closed: false, open_time: "08:00", close_time: "21:30" };
  assert.ok(validateFieldForm(form).schedules);
});

test("maps API field values to form fields", () => {
  const form = toFieldForm({
    name: "Arena",
    price_per_hour: 125000,
    facilities: ["Parkir", "Toilet"],
  }, []);
  assert.equal(form.price_per_hour, "125000");
  assert.equal(form.facilities, "Parkir, Toilet");
  assert.equal(form.schedules.length, 7);
});
