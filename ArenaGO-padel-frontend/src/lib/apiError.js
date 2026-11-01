export function getApiErrorMessage(status, contentType, body) {
  if (typeof body?.error === "string" && body.error) return body.error;
  if (typeof body?.message === "string" && body.message) return body.message;

  if (status >= 500 && !contentType.includes("json")) {
    return "Backend API tidak merespons. Jalankan `go run .` dari folder utama proyek, lalu coba lagi.";
  }

  return `Permintaan gagal (HTTP ${status}). Coba lagi atau periksa koneksi API.`;
}

export function getNetworkErrorMessage(error) {
  if (error instanceof TypeError) {
    return "Tidak dapat terhubung ke backend API. Pastikan backend berjalan dengan `go run .` di folder utama proyek.";
  }
  return error.message || "Tidak dapat menghubungi backend API.";
}
