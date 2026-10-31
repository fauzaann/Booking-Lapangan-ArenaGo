import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import Icon from "../components/ui/Icon";
import { api } from "../lib/api";
import { formatIDR } from "../lib/format";

export default function AdminCheckIn() {
  const [scanVersion, setScanVersion] = useState(0);
  const [cameraReady, setCameraReady] = useState(false);
  const [processing, setProcessing] = useState(false);
  const [booking, setBooking] = useState(null);
  const [error, setError] = useState("");
  const scannerRef = useRef(null);
  const handledRef = useRef(false);

  const stopScanner = useCallback(() => {
    const scanner = scannerRef.current;
    if (scanner?.isScanning) {
      scanner.stop().then(() => setCameraReady(false)).catch((stopError) => {
        console.error("QR scanner stop failed:", stopError);
      });
    }
  }, []);

  const handleDecoded = useCallback(async (value) => {
    if (handledRef.current) return;
    handledRef.current = true;
    setProcessing(true);
    setError("");

    try {
      const ticket = JSON.parse(value);
      if (
        ticket.type !== "ARENAGO_CHECK_IN" ||
        typeof ticket.booking_code !== "string" ||
        typeof ticket.check_in_token !== "string"
      ) {
        throw new Error("QR bukan e-tiket ArenaGo yang valid.");
      }

      const result = await api.adminCheckIn({
        booking_code: ticket.booking_code,
        check_in_token: ticket.check_in_token,
      });
      setBooking(result.data);
      stopScanner();
    } catch (requestError) {
      setError(requestError.message || "Gagal memvalidasi tiket.");
      stopScanner();
    } finally {
      setProcessing(false);
    }
  }, [stopScanner]);

  useEffect(() => {
    let active = true;
    let scanner;
    let startPromise;
    handledRef.current = false;
    import("html5-qrcode").then(({ Html5Qrcode }) => {
      if (!active) return;
      scanner = new Html5Qrcode("admin-checkin-reader");
      scannerRef.current = scanner;
      startPromise = scanner.start(
        { facingMode: "environment" },
        { fps: 10, qrbox: { width: 240, height: 240 } },
        (value) => {
          if (active) void handleDecoded(value);
        },
        () => {},
      ).then(() => {
        if (active) setCameraReady(true);
      }).catch((cameraError) => {
        if (active) setError(`Tidak dapat membuka kamera: ${cameraError.message || cameraError}`);
      });
    }).catch((loadError) => {
      if (active) setError(`Gagal memuat pemindai QR: ${loadError.message || loadError}`);
    });

    return () => {
      active = false;
      setCameraReady(false);
      if (!scanner) return;
      const clearScanner = () => {
        try {
          scanner.clear();
        } catch (clearError) {
          console.error("QR scanner cleanup failed:", clearError);
        }
      };
      const cleanup = () => {
        if (scanner.isScanning) {
          scanner.stop().then(clearScanner).catch((stopError) => {
            console.error("QR scanner stop failed:", stopError);
            clearScanner();
          });
        } else {
          clearScanner();
        }
      };
      if (startPromise) void startPromise.then(cleanup);
    };
  }, [handleDecoded, scanVersion]);

  function scanAgain() {
    setBooking(null);
    setError("");
    setCameraReady(false);
    setScanVersion((version) => version + 1);
  }

  return (
    <div className="mx-auto max-w-3xl px-5 py-8 sm:px-8">
      <Link to="/admin" className="mb-6 inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink">
        <Icon name="arrow_back" size={18} />
        Kembali ke dashboard
      </Link>

      <div className="mb-6">
        <p className="mb-2 text-xs font-bold uppercase tracking-[0.12em] text-champagne">ArenaGO Admin</p>
        <h1 className="font-display text-3xl">Check-in pemain</h1>
        <p className="mt-2 text-sm text-ink-muted">Pindai QR pada e-tiket pengguna untuk memvalidasi pembayaran dan mencatat kehadiran.</p>
      </div>

      <Card>
        {!booking && (
          <>
            <div id="admin-checkin-reader" className="mx-auto min-h-64 max-w-lg overflow-hidden rounded-md" />
            <p className="mt-4 text-center text-sm text-ink-muted">
              {processing ? "Memvalidasi tiket..." : cameraReady ? "Arahkan kamera ke QR e-tiket." : "Menyiapkan kamera..."}
            </p>
          </>
        )}

        {error && (
          <p role="alert" className="mt-4 rounded-md bg-red-50 px-4 py-3 text-sm text-red-700">
            {error}
          </p>
        )}

        {booking && (
          <div role="status" className="py-2">
            <div className="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-sage-tint text-sage">
              <Icon name="verified" size={30} />
            </div>
            <h2 className="text-center font-display text-2xl">Check-in berhasil</h2>
            <dl className="mx-auto mt-6 max-w-md space-y-3 rounded-md bg-surface px-5 py-4 text-sm">
              <InfoRow label="Kode booking" value={booking.booking_code} />
              <InfoRow label="Lapangan" value={booking.field_name || `Lapangan #${booking.field_id}`} />
              <InfoRow label="Tanggal" value={booking.booking_date} />
              <InfoRow label="Jam bermain" value={`${booking.start_time}–${booking.end_time}`} />
              <InfoRow label="Total dibayar" value={formatIDR(booking.total_price)} />
              <InfoRow label="Check-in tercatat" value={formatDateTime(booking.checked_in_at)} />
            </dl>
          </div>
        )}

        {(booking || error) && (
          <Button className="mt-5 w-full" onClick={scanAgain}>
            Pindai tiket berikutnya
          </Button>
        )}
      </Card>
    </div>
  );
}

function InfoRow({ label, value }) {
  return (
    <div className="flex justify-between gap-4">
      <dt className="text-ink-faint">{label}</dt>
      <dd className="text-right font-medium">{value}</dd>
    </div>
  );
}

function formatDateTime(value) {
  if (!value) return "-";
  return new Intl.DateTimeFormat("id-ID", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}
