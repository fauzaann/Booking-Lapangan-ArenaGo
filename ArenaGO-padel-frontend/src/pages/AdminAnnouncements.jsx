import { useState } from "react";
import Button from "../components/ui/Button";
import Input from "../components/ui/Input";
import { api } from "../lib/api";

export default function AdminAnnouncements() {
  const [form, setForm] = useState({ title: "", message: "" });
  const [result, setResult] = useState(null);
  const [error, setError] = useState("");
  const [sending, setSending] = useState(false);

  function update(field, value) {
    setForm((current) => ({ ...current, [field]: value }));
    setResult(null);
    setError("");
  }

  async function handleSubmit(event) {
    event.preventDefault();
    setSending(true);
    setError("");
    setResult(null);
    try {
      const response = await api.publishVenueAnnouncement(form);
      setResult(response.data);
      setForm({ title: "", message: "" });
    } catch (sendError) {
      setError(sendError.message);
    } finally {
      setSending(false);
    }
  }

  return (
    <div className="mx-auto max-w-3xl px-5 py-8 sm:px-8 lg:px-16">
      <p className="mb-2 text-xs font-bold uppercase tracking-[0.12em] text-champagne">ArenaGO Admin</p>
      <h1 className="font-display text-3xl sm:text-4xl">Pengumuman venue</h1>
      <p className="mb-8 mt-2 text-ink-muted">
        Pengumuman dikirim ke inbox seluruh pengguna. Pengguna yang menyetujui notifikasi WhatsApp juga akan menerima pesan WhatsApp.
      </p>

      <form onSubmit={handleSubmit} className="space-y-5 rounded-lg border border-border bg-surface-raised p-5">
        <Input
          label="Judul"
          value={form.title}
          maxLength={120}
          onChange={(event) => update("title", event.target.value)}
          required
        />
        <label className="block text-xs font-medium tracking-wide text-ink-muted">
          Isi pengumuman
          <textarea
            value={form.message}
            maxLength={1024}
            onChange={(event) => update("message", event.target.value)}
            required
            rows={5}
            className="mt-1.5 w-full rounded-md border border-border bg-surface-raised px-4 py-3 text-[15px] text-ink placeholder:text-ink-faint focus:border-onyx focus:outline-none focus:ring-4 focus:ring-onyx/5"
          />
        </label>
        {error && <p role="alert" className="text-sm text-red-600">{error}</p>}
        {result && (
          <div role="status" className="rounded-md border border-sage/30 bg-sage-tint p-4 text-sm text-ink">
            <p>Pengumuman diproses untuk {result.recipients} pengguna.</p>
            <p>
              WhatsApp: {result.whatsapp_queued ?? 0} pesan masuk antrean, {result.whatsapp_failed} gagal diantrekan dari {result.whatsapp_recipients} pengguna yang opt-in.
            </p>
            {result.in_app_failed > 0 && <p>Notifikasi in-app gagal: {result.in_app_failed}.</p>}
            {!result.in_app_enabled && <p>MongoDB tidak aktif; notifikasi in-app tidak tersedia.</p>}
            {!result.whatsapp_enabled && <p>Antrean WhatsApp tidak tersedia; pastikan koneksi MongoDB aktif.</p>}
            {result.whatsapp_queued > 0 && <p>Pengiriman WhatsApp membutuhkan proses notification worker yang aktif.</p>}
          </div>
        )}
        <Button type="submit" disabled={sending}>
          {sending ? "Mengirim..." : "Kirim pengumuman"}
        </Button>
      </form>
    </div>
  );
}
