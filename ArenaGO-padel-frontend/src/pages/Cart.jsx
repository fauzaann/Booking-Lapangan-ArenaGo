import { useState } from "react";
import { Link } from "react-router-dom";
import Button from "../components/ui/Button";
import Card from "../components/ui/Card";
import Icon from "../components/ui/Icon";
import { useCart } from "../context/CartContext";
import { api } from "../lib/api";
import { formatDateShort, formatIDR } from "../lib/format";

export default function Cart() {
  const { items, removeItem, clearCart } = useCart();
  const [conflicts, setConflicts] = useState({});
  const [error, setError] = useState("");
  const [checkingOut, setCheckingOut] = useState(false);

  const total = items.reduce((sum, item) => sum + item.totalPrice, 0);

  async function checkout() {
    setCheckingOut(true);
    setError("");
    setConflicts({});
    try {
      const result = await api.createCheckout({
        items: items.map((item) => ({
          item_id: item.itemId,
          field_id: Number(item.fieldId),
          booking_date: item.date,
          start_time: item.startTime,
          end_time: item.endTime,
        })),
      });
      const checkout = result.data;
      if (!checkout.payment_url) throw new Error("Link pembayaran tidak tersedia.");
      localStorage.setItem("arenago:pending-checkout", JSON.stringify({
        checkoutId: checkout.checkout_id,
      }));
      clearCart();
      window.location.assign(checkout.payment_url);
    } catch (requestError) {
      const itemConflicts = requestError.details?.conflicts;
      if (Array.isArray(itemConflicts)) {
        setConflicts(Object.fromEntries(itemConflicts.map((conflict) => [conflict.item_id, conflict.message])));
        setError("Ada jadwal yang sudah ter-booking. Keranjang Anda tetap tersimpan.");
      } else {
        setError(requestError.message);
      }
    } finally {
      setCheckingOut(false);
    }
  }

  return (
    <div className="mx-auto max-w-4xl px-5 py-8 sm:px-8 lg:px-16">
      <Link to="/" className="mb-6 inline-flex items-center gap-1.5 text-sm text-ink-muted hover:text-ink">
        <Icon name="arrow_back" size={18} />
        Kembali ke Eksplor
      </Link>
      <h1 className="mb-2 font-display text-3xl sm:text-4xl">Keranjang Booking</h1>
      <p className="mb-8 text-sm text-ink-muted">
        Jadwal belum dikunci sampai pembayaran selesai. Ketersediaan akan dicek kembali saat checkout.
      </p>

      {error && (
        <div role="alert" className="mb-5 rounded-md border border-terracotta/30 bg-terracotta/5 px-4 py-3 text-sm text-terracotta">
          {error}
        </div>
      )}

      {items.length === 0 ? (
        <Card className="py-10 text-center">
          <Icon name="shopping_basket" size={36} className="mb-3 text-ink-faint" />
          <p className="mb-4 text-ink-muted">Keranjang Anda masih kosong.</p>
          <Link to="/"><Button>Jelajahi Lapangan</Button></Link>
        </Card>
      ) : (
        <>
          <div className="space-y-3">
            {items.map((item) => (
              <Card key={item.itemId} className={conflicts[item.itemId] ? "border-terracotta" : ""}>
                <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                  <div>
                    <h2 className="font-semibold">{item.fieldName}</h2>
                    <p className="mt-1 text-sm text-ink-muted">
                      {formatDateShort(new Date(`${item.date}T00:00:00`))} · {item.startTime}–{item.endTime}
                    </p>
                    <p className="mt-1 text-sm font-medium">{formatIDR(item.totalPrice)}</p>
                    {conflicts[item.itemId] && (
                      <p className="mt-2 text-sm font-medium text-terracotta">{conflicts[item.itemId]}</p>
                    )}
                  </div>
                  <Button variant="secondary" onClick={() => removeItem(item.itemId)}>
                    Hapus
                  </Button>
                </div>
              </Card>
            ))}
          </div>
          <Card className="mt-6">
            <div className="mb-5 flex items-baseline justify-between">
              <span className="font-medium">Total pembayaran</span>
              <span className="font-display text-2xl">{formatIDR(total)}</span>
            </div>
            <div className="flex flex-col gap-3 sm:flex-row">
              <Button className="flex-1" disabled={!items.length || checkingOut} onClick={checkout}>
                {checkingOut ? "Memeriksa jadwal..." : "Checkout dan Bayar"}
              </Button>
              <Button variant="secondary" disabled={checkingOut} onClick={() => {
                clearCart();
                setConflicts({});
                setError("");
              }}>
                Kosongkan Keranjang
              </Button>
            </div>
          </Card>
        </>
      )}
    </div>
  );
}
