const apiBase = (process.env.API_URL || "http://localhost:8080/api/v1").replace(/\/$/, "");
const frontendUrl = (process.env.FRONTEND_URL || "http://localhost:5173").replace(/\/$/, "");
const webhookToken = process.env.XENDIT_WEBHOOK_TOKEN || "smoke-webhook-token";
const runPaymentFlow = process.env.SMOKE_PAYMENT !== "false";
const runWebhook = process.env.SMOKE_WEBHOOK !== "false";

async function request(url, options = {}) {
  const response = await fetch(url, {
    ...options,
    headers: { "Content-Type": "application/json", ...options.headers },
  });
  const body = await response.json().catch(() => ({}));
  return { response, body };
}

function assert(condition, message) {
  if (!condition) throw new Error(message);
}

function tomorrow(offset = 1) {
  const date = new Date();
  date.setDate(date.getDate() + offset);
  return date.toISOString().slice(0, 10);
}

async function main() {
  console.log(`Smoke test: frontend ${frontendUrl}`);
  const frontend = await fetch(frontendUrl);
  assert(frontend.ok, `frontend is not reachable: HTTP ${frontend.status}`);
  const html = await frontend.text();
  assert(html.includes('<div id="root">'), "frontend root element is missing");

  const health = await request(`${apiBase.replace(/\/api\/v1$/, "")}/health`);
  assert(health.response.ok && health.body.success, "backend health check failed");

  const suffix = Date.now().toString();
  const credentials = {
    name: `Smoke User ${suffix}`,
    email: `smoke-${suffix}@example.com`,
    password: "smoke123",
    phone: `0812${suffix.slice(-8)}`,
  };
  const register = await request(`${apiBase}/auth/register`, {
    method: "POST",
    body: JSON.stringify(credentials),
  });
  assert(register.response.status === 201, `register failed: ${register.response.status} ${register.body.message || ""}`);

  const login = await request(`${apiBase}/auth/login`, {
    method: "POST",
    body: JSON.stringify({ email: credentials.email, password: credentials.password }),
  });
  assert(login.response.ok && login.body.data?.token, "login did not return a JWT");
  const token = login.body.data.token;
  const authHeaders = { Authorization: `Bearer ${token}` };

  const fields = await request(`${apiBase}/fields?page=1&limit=50&status=ACTIVE`);
  assert(fields.response.ok && fields.body.data?.length, "no active field returned");
  const field = fields.body.data[0];

  const adminLogin = await request(`${apiBase}/auth/login`, {
    method: "POST",
    body: JSON.stringify({ email: "admin@example.com", password: "admin123" }),
  });
  assert(adminLogin.response.ok && adminLogin.body.data?.token, "seeded admin login failed");
  const dashboard = await request(`${apiBase}/admin/dashboard`, {
    headers: { Authorization: `Bearer ${adminLogin.body.data.token}` },
  });
  assert(dashboard.response.ok && dashboard.body.data, "admin dashboard endpoint failed");

  console.log(`Smoke passed: frontend, health, auth, fields, admin dashboard (${field.name})`);
  if (!runPaymentFlow) {
    console.log("Payment E2E skipped because SMOKE_PAYMENT=false");
    return;
  }

  let selectedSlot;
  let bookingDate;
  for (let offset = 1; offset <= 7 && !selectedSlot; offset += 1) {
    bookingDate = tomorrow(offset);
    const availability = await request(`${apiBase}/fields/${field.id}/availability?date=${bookingDate}`);
    if (!availability.response.ok) continue;
    selectedSlot = availability.body.data?.slots?.find((slot) => slot.available);
  }
  assert(selectedSlot, "no available slot found in the next seven days");

  const booking = await request(`${apiBase}/bookings`, {
    method: "POST",
    headers: authHeaders,
    body: JSON.stringify({
      field_id: Number(field.id),
      booking_date: bookingDate,
      start_time: selectedSlot.start_time,
      end_time: selectedSlot.end_time,
    }),
  });
  assert(booking.response.status === 201, `booking creation failed: ${booking.response.status} ${booking.body.message || ""} ${JSON.stringify(booking.body.error || {})}`);
  const created = booking.body.data;
  assert(created.booking_id && created.booking_code && created.total_price > 0, "booking response is incomplete");

  if (!runWebhook) {
    const detail = await request(`${apiBase}/bookings/${created.booking_id}`, { headers: authHeaders });
    assert(detail.response.ok && detail.body.data?.status === "WAITING_PAYMENT", `booking status after invoice creation: ${detail.body.data?.status}`);
    const payment = await request(`${apiBase}/bookings/${created.booking_id}/payment`, { headers: authHeaders });
    assert(payment.response.ok && payment.body.data?.status === "PENDING", `payment status after invoice creation: ${payment.body.data?.status}`);
    console.log(`Real Xendit invoice passed: ${created.booking_code} -> WAITING_PAYMENT / PENDING`);
    console.log(`Payment URL returned: ${created.payment_url}`);
    return;
  }

  const invalidWebhook = await request(`${apiBase}/payments/webhook`, {
    method: "POST",
    headers: { "x-callback-token": "invalid-token" },
    body: JSON.stringify({ external_id: created.booking_code, status: "PAID", amount: created.total_price }),
  });
  assert(invalidWebhook.response.status === 401, "invalid webhook token was accepted");

  const webhook = await request(`${apiBase}/payments/webhook`, {
    method: "POST",
    headers: { "x-callback-token": webhookToken },
    body: JSON.stringify({
      external_id: created.booking_code,
      status: "PAID",
      amount: created.total_price,
      paid_amount: created.total_price,
      payment_method: "SMOKE_TEST",
      payment_channel: "SMOKE_TEST",
    }),
  });
  assert(webhook.response.ok, `paid webhook failed: ${webhook.response.status} ${webhook.body.message || ""}`);

  const detail = await request(`${apiBase}/bookings/${created.booking_id}`, { headers: authHeaders });
  assert(detail.response.ok && detail.body.data?.status === "CONFIRMED", `booking did not become CONFIRMED: ${detail.body.data?.status}`);
  const payment = await request(`${apiBase}/bookings/${created.booking_id}/payment`, { headers: authHeaders });
  assert(payment.response.ok && payment.body.data?.status === "PAID", `payment did not become PAID: ${payment.body.data?.status}`);

  console.log(`Payment E2E passed: ${created.booking_code} -> PAID / CONFIRMED`);
}

main().catch((error) => {
  console.error(`Smoke test failed: ${error.message}`);
  process.exitCode = 1;
});
