import { useParams, Link } from "react-router-dom";
import Icon from "../components/ui/Icon";
import Card from "../components/ui/Card";
import StatusBadge from "../components/ui/StatusBadge";
import { transactions } from "../lib/mockData";
import { formatIDR } from "../lib/format";

export default function TransactionDetail() {
  const { id } = useParams();
  const txn = transactions.find((t) => t.id === id) ?? transactions[0];

  return (
    <div className="px-5 py-8 sm:px-8 lg:px-16">
      <Link
        to="/admin/transaksi"
        className="mb-6 inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink"
      >
        <Icon name="arrow_back" size={18} />
        Kembali ke Riwayat
      </Link>

      <div className="mb-8 flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="mb-1 text-xs uppercase tracking-[0.12em] text-ink-faint">
            {txn.id}
          </p>
          <h1 className="font-display text-3xl">{formatIDR(txn.amount)}</h1>
        </div>
        <StatusBadge status={txn.status} className="mt-1" />
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_1fr]">
        <div className="space-y-6">
          <Card>
            <h2 className="mb-4 font-display text-xl">Detail Booking</h2>
            <dl className="space-y-3 text-sm">
              <Row label="Booking ID" value={txn.bookingId} />
              <Row label="Pelanggan" value={txn.customer} />
              <Row label="Lapangan" value={txn.court} />
              <Row label="Metode Pembayaran" value={txn.method} />
              <Row label="Dibuat" value={new Date(txn.createdAt).toLocaleString("id-ID")} />
              <Row
                label="Dibayar"
                value={txn.paidAt ? new Date(txn.paidAt).toLocaleString("id-ID") : "—"}
              />
            </dl>
          </Card>

          <Card>
            <h2 className="mb-4 font-display text-xl">Payload Xendit (mock)</h2>
            <pre className="overflow-x-auto rounded-md bg-onyx px-4 py-4 text-xs leading-relaxed text-canvas/90">
{JSON.stringify(txn.rawPayload, null, 2)}
            </pre>
          </Card>
        </div>

        <Card>
          <h2 className="mb-5 font-display text-xl">Log Webhook</h2>
          <ol className="space-y-6">
            {txn.webhookEvents.map((event, i) => (
              <li key={i} className="relative flex gap-4 pl-1">
                <div className="flex flex-col items-center">
                  <span className="flex h-7 w-7 items-center justify-center rounded-full bg-sage-tint text-sage">
                    <Icon name="bolt" size={14} />
                  </span>
                  {i < txn.webhookEvents.length - 1 && (
                    <span className="mt-1 h-full w-px flex-1 bg-border" />
                  )}
                </div>
                <div className="pb-2">
                  <p className="text-sm font-medium">{event.event}</p>
                  <p className="text-xs text-ink-faint">
                    {new Date(event.at).toLocaleString("id-ID")} · HTTP {event.status}
                  </p>
                </div>
              </li>
            ))}
            {txn.status === "pending" && (
              <li className="flex items-center gap-4 pl-1 text-ink-faint">
                <span className="flex h-7 w-7 items-center justify-center rounded-full border border-dashed border-border">
                  <Icon name="hourglass_empty" size={14} />
                </span>
                <p className="text-sm">Menunggu event lanjutan dari Xendit...</p>
              </li>
            )}
          </ol>
        </Card>
      </div>
    </div>
  );
}

function Row({ label, value }) {
  return (
    <div className="flex justify-between gap-4 border-b border-border pb-3 last:border-b-0 last:pb-0">
      <dt className="text-ink-faint">{label}</dt>
      <dd className="text-right font-medium">{value}</dd>
    </div>
  );
}
