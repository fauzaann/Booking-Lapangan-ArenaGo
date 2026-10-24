import assert from "node:assert/strict";
import test from "node:test";
import { toggleContiguousSlot } from "../src/lib/bookingSlots.js";

const slots = [
  { time: "10:00", status: "available" },
  { time: "11:00", status: "available" },
  { time: "12:00", status: "available" },
  { time: "13:00", status: "booked" },
  { time: "14:00", status: "available" },
];

test("extends a booking through adjacent available hours", () => {
  let selected = toggleContiguousSlot(slots, [], "11:00");
  selected = toggleContiguousSlot(slots, selected, "10:00");
  selected = toggleContiguousSlot(slots, selected, "12:00");

  assert.deepEqual(selected, ["10:00", "11:00", "12:00"]);
});

test("starts a new range instead of creating a gap", () => {
  const selected = toggleContiguousSlot(slots, ["10:00", "11:00"], "14:00");

  assert.deepEqual(selected, ["14:00"]);
});

test("does not add a booked hour to the selected range", () => {
  const selected = toggleContiguousSlot(slots, ["11:00", "12:00"], "13:00");

  assert.deepEqual(selected, ["11:00", "12:00"]);
});

test("removes an end hour without breaking a multi-hour range", () => {
  const selected = toggleContiguousSlot(slots, ["10:00", "11:00", "12:00"], "12:00");

  assert.deepEqual(selected, ["10:00", "11:00"]);
});
