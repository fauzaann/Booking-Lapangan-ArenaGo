import assert from "node:assert/strict";
import test from "node:test";
import { isAvailabilityQuestion, parseAvailabilityRequest } from "./assistantAvailability.js";

const now = new Date(2026, 9, 31, 12);

test("recognizes availability questions", () => {
  assert.equal(isAvailabilityQuestion("Lapangan apa yang kosong besok?"), true);
  assert.equal(isAvailabilityQuestion("Berapa harga lapangan?"), false);
});

test("parses a relative date and a requested hour", () => {
  const result = parseAvailabilityRequest([
    { from: "user", text: "Lapangan apa yang tersedia besok jam 19?" },
  ], now);

  assert.deepEqual(result, {
    date: "2026-11-01",
    slots: ["19:00"],
    needsTimeClarification: false,
  });
});

test("parses an Indonesian evening range and an explicit numeric date", () => {
  const result = parseAvailabilityRequest([
    { from: "user", text: "Cek slot kosong tanggal 02/11/2026 jam 7 sampai 9 malam" },
  ], now);

  assert.deepEqual(result, {
    date: "2026-11-02",
    slots: ["19:00", "20:00"],
    needsTimeClarification: false,
  });
});

test("parses a textual date and a weekday", () => {
  assert.equal(parseAvailabilityRequest([
    { from: "user", text: "Jadwal kosong 5 November jam 14:00" },
  ], now).date, "2026-11-05");
  assert.equal(parseAvailabilityRequest([
    { from: "user", text: "Lapangan kosong hari Senin pukul 10" },
  ], now).date, "2026-11-02");
});

test("uses earlier user context only when the question refers to it", () => {
  const history = [
    { from: "user", text: "Besok jam 19 lapangan apa yang kosong?" },
    { from: "assistant", text: "Aku cek dulu." },
    { from: "user", text: "Bagaimana dengan jam tersebut?" },
  ];

  assert.deepEqual(parseAvailabilityRequest(history, now), {
    date: "2026-11-01",
    slots: ["19:00"],
    needsTimeClarification: false,
  });
  assert.equal(parseAvailabilityRequest([
    { from: "user", text: "Besok jam 19 lapangan apa yang kosong?" },
    { from: "user", text: "Lapangan apa yang kosong Jumat?" },
  ], now).slots, null);
});

test("ignores malformed dates and asks for missing dates or referenced times", () => {
  assert.equal(parseAvailabilityRequest([
    { from: "user", text: "Cek yang kosong tanggal 31/02/2026" },
  ], now).date, null);
  assert.equal(parseAvailabilityRequest([
    { from: "user", text: "Jam tersebut masih tersedia?" },
  ], now).needsTimeClarification, true);
});

test("does not mistake the day number in a date for a requested hour", () => {
  const result = parseAvailabilityRequest([
    { from: "user", text: "Lapangan apa yang kosong?" },
    { from: "assistant", text: "Tanggal berapa?", pendingAvailabilityQuestion: "Lapangan apa yang kosong?" },
    { from: "user", text: "15 November 2026" },
  ], now);

  assert.equal(result.date, "2026-11-15");
  assert.equal(result.slots, null);
});
