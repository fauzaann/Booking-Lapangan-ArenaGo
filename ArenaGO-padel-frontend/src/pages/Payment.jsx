import { useEffect, useState } from "react";
import { useLocation, useNavigate, Link } from "react-router-dom";
import Icon from "../components/ui/Icon";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import BookingSummary from "../components/booking/BookingSummary";
import { paymentMethods, sampleBooking } from "../lib/mockData";
import { bookingPriceBreakdown } from "../lib/mockData";
import { formatIDR } from "../lib/format";

const COUNTDOWN_SECONDS = 15 * 60;

export default function Payment() {
  const { state } = useLocation();
  const navigate = useNavigate();
  const booking = state?.booking ?? sampleBooking;

  const [method, setMethod] = useState("qris");
  const [seconds, setSeconds] = useState(COUNTDOWN_SECONDS);
  const [status, setStatus] = useState("idle"); // idle | processing | success

  useEffect(() => {
    if (status !== "idle") return;
    const timer = setInterval(() => setSeconds((s) => Math.max(0, s - 1)), 1000);
    return () => clearInterval(timer);
  }, [status]);

  const total = bookingPriceBreakdown(booking).find((b) => b.label === "__total__").amount;
  const mm = String(Math.floor(seconds / 60)).padStart(2, "0");
  const ss = String(seconds % 60).padStart(2, "0");

  function pay() {
    setStatus("processing");
    setTimeout(() => {
      setStatus("success");
      setTimeout(() => {
        navigate("/e-tiket", { state: { booking, method } });
      }, 900);
    }, 1400);
  }

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <Link
        to="/konfirmasi"
        state={{ booking }}
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink"
      >
        <Icon name="arrow_back" size={18} />
        Kembali
      </Link>

      <div className="grid grid-cols-1 gap-10 lg:grid-cols-[1fr_380px]">
        <div>
          <div className="mb-6 flex items-center justify-between">
            <h1 className="font-display text-3xl sm:text-4xl">Pembayaran</h1>
            <div className="flex items-center gap-2 rounded-full border border-champagne/50 bg-champagne-tint px-4 py-2 text-sm">
              <Icon name="schedule" size={16} className="text-champagne" />
              <span className="font-mono font-semibold tabular-nums">{mm}:{ss}</span>
            </div>
          </div>
          <p className="mb-8 text-ink-muted">
            Diproses secara aman melalui payment gateway Xendit. Pilih salah satu
            metode di bawah untuk menyelesaikan pembayaran.
          </p>

          <div className="space-y-6">
            {paymentMethods.map((group) => (
              <div key={group.group}>
                <h3 className="mb-3 text-xs font-bold uppercase tracking-[0.12em] text-ink-faint">
                  {group.group}
                </h3>
                <div className="space-y-2.5">
                  {group.items.map((item) => {
                    const isSelected = method === item.id;
                    return (
                      <button
                        key={item.id}
                        onClick={() => setMethod(item.id)}
                        className={`flex w-full items-center justify-between rounded-md border bg-surface-raised px-5 py-4 text-left transition-all duration-150 ${
                          isSelected
                            ? "border-[1.5px] border-onyx"
                            : "border-border hover:border-ink-muted"
                        }`}
                      >
                        <span className="flex items-center gap-3">
                          <Icon name={item.icon} className="text-ink-muted" />
                          <span className="text-sm font-medium">{item.name}</span>
                        </span>
                        {isSelected && (
                          <span className="flex h-5 w-5 items-center justify-center rounded-full bg-sage text-canvas">
                            <Icon name="check" size={14} />
                          </span>
                        )}
                      </button>
                    );
                  })}
                </div>
              </div>
            ))}
          </div>

          {method === "qris" && (
            <Card className="mt-6 flex flex-col items-center gap-4 py-8 text-center">
              <QrPlaceholder />
              <p className="text-sm text-ink-muted">
                Scan kode QR di atas menggunakan aplikasi GoPay, OVO, ShopeePay, atau Dana.
              </p>
            </Card>
          )}
        </div>

        <div className="lg:sticky lg:top-24 lg:self-start">
          <Card>
            <BookingSummary booking={booking} showBreakdown={false} />
            <Button
              className="mt-6 w-full"
              onClick={pay}
              disabled={status !== "idle"}
            >
              {status === "idle" && `Bayar ${formatIDR(total)}`}
              {status === "processing" && "Memproses..."}
              {status === "success" && "Pembayaran Berhasil"}
            </Button>
            <p className="mt-3 text-center text-[11px] text-ink-faint">
              Simulasi tampilan pembayaran — tidak ada transaksi nyata yang diproses.
            </p>
          </Card>
        </div>
      </div>
    </div>
  );
}

function QrPlaceholder() {
  // Purely decorative mock QR pattern — not a scannable code.
  const cells = Array.from({ length: 49 }).map((_, i) => (i * 7 + 3) % 5 === 0);
  return (
    <div className="grid h-40 w-40 grid-cols-7 gap-1 rounded-md border border-border bg-white p-3">
      {cells.map((filled, i) => (
        <span key={i} className={`rounded-[2px] ${filled ? "bg-onyx" : "bg-transparent"}`} />
      ))}
    </div>
  );
}
