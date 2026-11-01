import test from "node:test";
import assert from "node:assert/strict";

import { getApiErrorMessage, getNetworkErrorMessage } from "./apiError.js";

test("uses a readable backend message when the API returns JSON", () => {
  assert.equal(getApiErrorMessage(401, "application/json", {
    message: "authorization header is required",
  }), "authorization header is required");
});

test("explains when the development proxy cannot reach the backend", () => {
  assert.match(getApiErrorMessage(500, "text/html", undefined), /go run \./);
});

test("reports an HTTP status when the server omits its error message", () => {
  assert.equal(getApiErrorMessage(502, "application/json", {}), "Permintaan gagal (HTTP 502). Coba lagi atau periksa koneksi API.");
});

test("gives a specific instruction for a failed fetch connection", () => {
  assert.match(getNetworkErrorMessage(new TypeError("fetch failed")), /go run \./);
});
