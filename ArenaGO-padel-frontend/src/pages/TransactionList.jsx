import { useState } from "react";
import { Link } from "react-router-dom";
import Icon from "../components/ui/Icon";
import StatusBadge from "../components/ui/StatusBadge";
import Tabs from "../components/ui/Tabs";
import { transactions } from "../lib/mockData";
import { formatIDR } from "../lib/format";

const FILTERS = [
  { value: "all", label: "Semua" },
  { value: "success", label: "Berhasil" },
  { value: "pending", label: "Menunggu" },
  { value: "failed", label: "Gagal" },
];

export default function TransactionList() {
  const [filter, setFilter] = useState("all");
  const filtered = transactions.filter((t) => filter === "all" || t.status === filter);

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <h1 className="mb-2 font-display text-3xl sm:text-4xl">Riwayat Transaksi</h1>
      <p className="mb-8 text-ink-muted">
        Pantau status pembayaran dan detail log webhook Xendit untuk setiap booking.
      </p>

      <Tabs tabs={FILTERS} active={filter} onChange={setFilter} className="mb-6" />

      {filtered.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed border-border py-20 text-center">
          <Icon name="receipt_long" size={32} className="mb-3 text-ink-faint" />
          <p className="mb-1 font-display text-xl">Belum ada transaksi</p>
          <p className="text-sm text-ink-muted">Transaksi dengan status ini belum tersedia.</p>
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <div className="hidden grid-cols-[1.2fr_1fr_1fr_0.8fr_1fr_40px] gap-4 border-b border-border bg-surface px-6 py-3 text-xs font-semibold uppercase tracking-wide text-ink-faint md:grid">
            <span>Transaksi</span>
            <span>Lapangan</span>
            <span>Metode</span>
            <span>Jumlah</span>
            <span>Status</span>
            <span />
          </div>
          {filtered.map((t) => (
            <Link
              key={t.id}
              to={`/admin/transaksi/${t.id}`}
              className="grid grid-cols-2 gap-3 border-b border-border px-6 py-4 last:border-b-0 hover:bg-surface/60 md:grid-cols-[1.2fr_1fr_1fr_0.8fr_1fr_40px] md:items-center md:gap-4"
            >
              <div>
                <p className="text-sm font-semibold">{t.id}</p>
                <p className="text-xs text-ink-faint">{t.customer}</p>
              </div>
              <p className="text-sm text-ink-muted">{t.court}</p>
              <p className="text-sm text-ink-muted">{t.method}</p>
              <p className="text-sm font-medium">{formatIDR(t.amount)}</p>
              <div>
                <StatusBadge status={t.status} />
              </div>
              <Icon name="chevron_right" className="hidden text-ink-faint md:block" />
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
