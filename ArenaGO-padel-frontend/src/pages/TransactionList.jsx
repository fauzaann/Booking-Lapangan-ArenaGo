import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import Icon from "../components/ui/Icon";
import StatusBadge from "../components/ui/StatusBadge";
import Tabs from "../components/ui/Tabs";
import { api } from "../lib/api";
import { formatIDR } from "../lib/format";

const FILTERS = [
  { value: "all", label: "Semua" },
  { value: "PAID", label: "Berhasil" },
  { value: "PENDING", label: "Menunggu" },
  { value: "FAILED", label: "Gagal" },
  { value: "EXPIRED", label: "Expired" },
];

export default function TransactionList() {
  const [filter, setFilter] = useState("all");
  const [payments, setPayments] = useState([]);
  const [meta, setMeta] = useState(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    setError("");
    const params = { page: "1", limit: "50" };
    if (filter !== "all") params.status = filter;
    api.adminPayments(params)
      .then((result) => {
        setPayments(result.data);
        setMeta(result.meta);
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  }, [filter]);

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <h1 className="mb-2 font-display text-3xl sm:text-4xl">Riwayat Transaksi</h1>
      <p className="mb-8 text-ink-muted">
        Pantau status pembayaran dan detail log webhook Xendit untuk setiap booking.
      </p>

      <Tabs tabs={FILTERS} active={filter} onChange={setFilter} className="mb-6" />

      {error ? <div className="rounded-lg border border-dashed border-border py-20 text-center text-sm text-red-600">{error}</div> : null}
      {loading ? <p className="py-16 text-center text-sm text-ink-muted">Memuat transaksi...</p> : null}
      {!loading && !error && payments.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-20 text-center">
          <Icon name="receipt_long" size={32} className="mb-3 text-ink-faint" />
          <p className="mb-1 font-display text-xl">Belum ada transaksi</p>
          <p className="text-sm text-ink-muted">Transaksi dengan status ini belum tersedia.</p>
        </div>
      ) : !loading && !error ? (
        <div className="overflow-hidden rounded-lg border border-border">
          <div className="hidden grid-cols-[1.2fr_1fr_1fr_0.8fr_1fr_40px] gap-4 border-b border-border bg-surface px-6 py-3 text-xs font-semibold uppercase tracking-wide text-ink-faint md:grid">
            <span>Transaksi</span>
            <span>Lapangan</span>
            <span>Metode</span>
            <span>Jumlah</span>
            <span>Status</span>
            <span />
          </div>
          {payments.map((payment) => (
            <Link
              key={payment.id}
              to={`/admin/transaksi/${payment.booking_id}`}
              className="grid grid-cols-2 gap-3 border-b border-border px-6 py-4 last:border-b-0 hover:bg-surface/60 md:grid-cols-[1.2fr_1fr_1fr_0.8fr_1fr_40px] md:items-center md:gap-4"
            >
              <div>
                <p className="text-sm font-semibold">{payment.booking_code || payment.external_id}</p>
                <p className="text-xs text-ink-faint">{payment.customer_name || "Pelanggan"}</p>
              </div>
              <p className="text-sm text-ink-muted">{payment.field_name || `Lapangan #${payment.booking_id}`}</p>
              <p className="text-sm text-ink-muted">{payment.payment_method || payment.payment_channel || "Belum dibayar"}</p>
              <p className="text-sm font-medium">{formatIDR(payment.amount)}</p>
              <div>
                <StatusBadge status={mapPaymentStatus(payment.status)} label={formatPaymentStatus(payment.status)} />
              </div>
              <Icon name="chevron_right" className="hidden text-ink-faint md:block" />
            </Link>
          ))}
          {meta && meta.total_items > payments.length ? <p className="mt-4 text-right text-xs text-ink-faint">Menampilkan {payments.length} dari {meta.total_items} transaksi</p> : null}
        </div>
      ) : null}
    </div>
  );
}

function mapPaymentStatus(status) {
  if (status === "PAID") return "success";
  if (status === "PENDING") return "pending";
  return "failed";
}

function formatPaymentStatus(status) {
  return { PAID: "Berhasil", PENDING: "Menunggu Pembayaran", FAILED: "Gagal", EXPIRED: "Expired" }[status] || status;
}
