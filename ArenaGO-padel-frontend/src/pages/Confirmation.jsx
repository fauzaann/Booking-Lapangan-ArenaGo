import { useState } from "react";
import { useLocation, useNavigate, Link } from "react-router-dom";
import Icon from "../components/ui/Icon";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import Input from "../components/ui/Input";
import BookingSummary from "../components/booking/BookingSummary";
import { sampleBooking } from "../lib/mockData";
import { api } from "../lib/api";
import { formatDateInput } from "../lib/format";

export default function Confirmation() {
  const { state } = useLocation();
  const navigate = useNavigate();

  const booking = state?.court
    ? {
        ...sampleBooking,
        court: state.court,
        date: state.date,
        slots: state.slots?.length ? state.slots : sampleBooking.slots,
        totalOverride: state.court.pricePerHour * (state.slots?.length || sampleBooking.slots.length),
      }
    : sampleBooking;

  const [form, setForm] = useState(booking.customer);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  function update(field, value) {
    setForm((prev) => ({ ...prev, [field]: value }));
  }

  const activeBooking = { ...booking, customer: form, rental: { rackets: 0, balls: 0 } };

  async function createBooking() {
    setSubmitting(true);
    setError("");
    try {
      const result = await api.createBooking({
        field_id: Number(booking.court.id),
        booking_date: formatDateInput(booking.date),
        start_time: booking.slots[0],
        end_time: addHour(booking.slots[booking.slots.length - 1]),
      });
      localStorage.setItem("arenago:pending-payment", JSON.stringify({
        bookingId: result.data.booking_id,
        bookingCode: result.data.booking_code,
      }));
      navigate("/pembayaran", { state: { booking: activeBooking, created: result.data } });
    } catch (err) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <Link
        to={`/jadwal/${booking.court.id}`}
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink"
      >
        <Icon name="arrow_back" size={18} />
        Ubah Jadwal
      </Link>

      <div className="grid grid-cols-1 gap-10 lg:grid-cols-[1fr_380px]">
        <div className="space-y-8">
          <div>
            <h1 className="mb-2 font-display text-3xl sm:text-4xl">Konfirmasi Booking</h1>
            <p className="text-ink-muted">
              Lengkapi data pemesan dan tambahan sewa sebelum melanjutkan ke pembayaran.
            </p>
          </div>

          <Card>
            <h2 className="mb-4 font-display text-xl">Data Pemesan</h2>
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <Input
                label="Nama Lengkap"
                value={form.name}
                onChange={(e) => update("name", e.target.value)}
              />
              <Input
                label="Nomor WhatsApp"
                value={form.phone}
                onChange={(e) => update("phone", e.target.value)}
              />
              <Input
                label="Email"
                className="sm:col-span-2"
                value={form.email}
                onChange={(e) => update("email", e.target.value)}
              />
            </div>
          </Card>

          <Card className="border-champagne/40 bg-champagne-tint/40">
            <div className="flex gap-3">
              <Icon name="info" className="mt-0.5 shrink-0 text-champagne" />
              <p className="text-sm text-ink-muted">
                E-tiket akan dikirim ke email dan WhatsApp Anda setelah pembayaran
                berhasil. Tunjukkan QR pada e-tiket ke staf lapangan saat check-in.
              </p>
            </div>
          </Card>
        </div>

        <div className="lg:sticky lg:top-24 lg:self-start">
          <Card>
            <BookingSummary booking={activeBooking} />
            {error && <p className="mt-4 text-sm text-red-600">{error}</p>}
            <Button className="mt-6 w-full" onClick={createBooking} disabled={submitting}>
              {submitting ? "Membuat booking..." : "Lanjut ke Pembayaran"}
            </Button>
          </Card>
        </div>
      </div>
    </div>
  );
}

function addHour(time) {
  const [hour, minute] = time.split(":").map(Number);
  return `${String((hour + 1) % 24).padStart(2, "0")}:${String(minute).padStart(2, "0")}`;
}

