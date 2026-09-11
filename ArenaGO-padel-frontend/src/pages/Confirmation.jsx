import { useState } from "react";
import { useLocation, useNavigate, Link } from "react-router-dom";
import Icon from "../components/ui/Icon";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import Input from "../components/ui/Input";
import BookingSummary from "../components/booking/BookingSummary";
import { courts, upcomingDays, sampleBooking } from "../lib/mockData";

export default function Confirmation() {
  const { state } = useLocation();
  const navigate = useNavigate();

  const booking = state
    ? {
        ...sampleBooking,
        court: courts.find((c) => c.id === state.courtId) ?? sampleBooking.court,
        date: upcomingDays[state.dayIndex] ?? sampleBooking.date,
        slots: state.slots?.length ? state.slots : sampleBooking.slots,
      }
    : sampleBooking;

  const [form, setForm] = useState(booking.customer);
  const [rentals, setRentals] = useState(booking.rental);

  function update(field, value) {
    setForm((prev) => ({ ...prev, [field]: value }));
  }

  const activeBooking = { ...booking, customer: form, rental: rentals };

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

          <Card>
            <h2 className="mb-4 font-display text-xl">Sewa Tambahan</h2>
            <div className="space-y-4">
              <RentalRow
                icon="sports_tennis"
                label="Raket Padel"
                sub="IDR 75.000 / raket"
                value={rentals.rackets}
                onChange={(v) => setRentals((p) => ({ ...p, rackets: v }))}
              />
              <RentalRow
                icon="fiber_manual_record"
                label="Bola Padel"
                sub="IDR 35.000 / tube"
                value={rentals.balls}
                onChange={(v) => setRentals((p) => ({ ...p, balls: v }))}
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
            <Button
              className="mt-6 w-full"
              onClick={() =>
                navigate("/pembayaran", { state: { booking: activeBooking } })
              }
            >
              Lanjut ke Pembayaran
            </Button>
          </Card>
        </div>
      </div>
    </div>
  );
}

function RentalRow({ icon, label, sub, value, onChange }) {
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-3">
        <span className="flex h-10 w-10 items-center justify-center rounded-full bg-surface">
          <Icon name={icon} size={18} />
        </span>
        <div>
          <p className="text-sm font-medium">{label}</p>
          <p className="text-xs text-ink-faint">{sub}</p>
        </div>
      </div>
      <div className="flex items-center gap-3">
        <button
          onClick={() => onChange(Math.max(0, value - 1))}
          className="flex h-8 w-8 items-center justify-center rounded-full border border-border hover:border-onyx"
          aria-label={`Kurangi ${label}`}
        >
          <Icon name="remove" size={16} />
        </button>
        <span className="w-4 text-center text-sm font-medium">{value}</span>
        <button
          onClick={() => onChange(value + 1)}
          className="flex h-8 w-8 items-center justify-center rounded-full border border-border hover:border-onyx"
          aria-label={`Tambah ${label}`}
        >
          <Icon name="add" size={16} />
        </button>
      </div>
    </div>
  );
}
