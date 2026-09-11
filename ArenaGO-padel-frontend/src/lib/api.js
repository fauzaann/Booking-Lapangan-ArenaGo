const API_BASE_URL = (import.meta.env.VITE_API_URL || "http://localhost:8080/api/v1").replace(/\/$/, "");

async function request(path, options = {}) {
  const token = localStorage.getItem("arenago:token");
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options.headers,
    },
  });

  const body = await response.json().catch(() => ({}));
  if (!response.ok || body.success === false) {
    const message = typeof body.error === "string" ? body.error : body.message;
    throw new Error(message || "Permintaan gagal diproses.");
  }
  return body;
}

function mapField(field, index = 0) {
  return {
    ...field,
    id: String(field.id),
    pricePerHour: field.price_per_hour,
    imageUrl: field.image_url || "",
    tags: field.facilities || [],
    type: field.type.replaceAll("_", " "),
    surface: field.description || "Lapangan premium",
    rating: null,
    reviews: 0,
    gradient: [
      "from-[#2D4B3E] to-[#14171A]",
      "from-[#1E3A5F] to-[#14171A]",
      "from-[#725B38] to-[#14171A]",
      "from-[#3A4B4B] to-[#14171A]",
    ][index % 4],
    vip: index === 0,
  };
}

export const api = {
  login: (payload) => request("/auth/login", { method: "POST", body: JSON.stringify(payload) }),
  register: (payload) => request("/auth/register", { method: "POST", body: JSON.stringify(payload) }),
  me: () => request("/me"),
  fields: async (params = {}) => {
    const query = new URLSearchParams({ page: "1", limit: "50", status: "ACTIVE", ...params });
    const result = await request(`/fields?${query}`);
    return { ...result, data: result.data.map(mapField) };
  },
  field: async (id) => {
    const result = await request(`/fields/${id}`);
    return { ...result, data: mapField(result.data) };
  },
  availability: (id, date) => request(`/fields/${id}/availability?date=${encodeURIComponent(date)}`),
  createBooking: (payload) => request("/bookings", { method: "POST", body: JSON.stringify(payload) }),
  booking: (id) => request(`/bookings/${id}`),
  bookingPayment: (id) => request(`/bookings/${id}/payment`),
  bookings: (params = {}) => request(`/bookings?${new URLSearchParams(params)}`),
  adminDashboard: () => request("/admin/dashboard"),
  adminBookings: (params = {}) => request(`/admin/bookings?${new URLSearchParams({ page: "1", limit: "8", ...params })}`),
  adminFields: () => request("/admin/fields?page=1&limit=50"),
  adminPayments: (params = {}) => request(`/admin/payments?${new URLSearchParams({ page: "1", limit: "50", ...params })}`),
  assistantChat: (payload) => request("/assistant/chat", { method: "POST", body: JSON.stringify(payload) }),
};

export function saveSession(auth) {
  localStorage.setItem("arenago:token", auth.token);
  localStorage.setItem("arenago:session", JSON.stringify(auth.user));
}

export function clearSession() {
  localStorage.removeItem("arenago:token");
  localStorage.removeItem("arenago:session");
}