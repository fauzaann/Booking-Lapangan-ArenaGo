const STATUS_STYLES = {
  available: { text: "text-sage", bg: "bg-sage-tint", label: "Tersedia", dot: "bg-sage" },
  success: { text: "text-sage", bg: "bg-sage-tint", label: "Berhasil", dot: "bg-sage" },
  confirmed: { text: "text-onyx", bg: "bg-onyx/[0.06]", label: "Terkonfirmasi", dot: "bg-onyx" },
  pending: { text: "text-[#8C6B38]", bg: "bg-champagne/[0.15]", label: "Menunggu Pembayaran", dot: "bg-champagne" },
  failed: { text: "text-terracotta", bg: "bg-terracotta/[0.08]", label: "Gagal", dot: "bg-terracotta" },
  booked: { text: "text-slate", bg: "bg-slate/[0.12]", label: "Terisi", dot: "bg-slate" },
};

export default function StatusBadge({ status, label, className = "" }) {
  const style = STATUS_STYLES[status] || STATUS_STYLES.confirmed;
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-3 py-1 text-xs font-semibold tracking-wide ${style.bg} ${style.text} ${className}`}
    >
      <span className={`h-1.5 w-1.5 rounded-full ${style.dot}`} />
      {label || style.label}
    </span>
  );
}
