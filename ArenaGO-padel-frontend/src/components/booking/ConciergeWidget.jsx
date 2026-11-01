import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import Icon from "../ui/Icon";
import { conciergeSeedMessages, conciergeQuickActions } from "../../lib/mockData";
import { api } from "../../lib/api";
import { formatIDR } from "../../lib/format";
import { isAvailabilityQuestion, parseAvailabilityRequest } from "../../lib/assistantAvailability";

export default function ConciergeWidget() {
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [messages, setMessages] = useState(conciergeSeedMessages);
  const [draft, setDraft] = useState("");
  const [loading, setLoading] = useState(false);
  const messagesEndRef = useRef(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, loading]);

  async function sendMessage(text) {
    const value = text ?? draft;
    if (!value.trim()) return;
    const history = messages.map((message) => ({
      role: message.from === "user" ? "user" : "assistant",
      content: message.text,
    }));
    const lastMessage = messages.at(-1);
    const pendingAvailability = lastMessage?.from === "assistant"
      ? lastMessage.pendingAvailabilityQuestion
      : null;
    setMessages((prev) => [...prev, { from: "user", text: value }]);
    setDraft("");
    setLoading(true);
    try {
      if (isAvailabilityQuestion(value) || pendingAvailability) {
        const availabilityMessage = pendingAvailability
          ? `${pendingAvailability} ${value}`
          : value;
        const request = parseAvailabilityRequest([
          ...messages,
          { from: "user", text: availabilityMessage },
        ]);
        if (!request.date) {
          setMessages((prev) => [...prev, {
            from: "assistant",
            text: "Bisa. Kamu ingin cek ketersediaan untuk tanggal berapa? Contoh: besok atau 15 November.",
            pendingAvailabilityQuestion: availabilityMessage,
          }]);
          return;
        }
        if (request.needsTimeClarification) {
          setMessages((prev) => [...prev, {
            from: "assistant",
            text: "Jam yang dimaksud pukul berapa? Sebutkan tanggal dan jamnya, misalnya besok pukul 19.00.",
            pendingAvailabilityQuestion: availabilityMessage,
          }]);
          return;
        }

        const fieldsResult = await api.fields({ page: "1", limit: "50", status: "ACTIVE" });
        const availability = await Promise.all(fieldsResult.data.map(async (field) => {
          const result = await api.availability(field.id, request.date);
          const freeSlots = result.data.slots.filter((slot) => slot.available);
          const matchingSlots = request.slots
            ? freeSlots.filter((slot) => request.slots.includes(slot.start_time))
            : freeSlots;
          if (request.slots && matchingSlots.length !== request.slots.length) return null;
          return matchingSlots.length > 0 ? { field, slots: matchingSlots } : null;
        }));
        const availableFields = availability.filter(Boolean);
        const timeDescription = request.slots
          ? `pukul ${request.slots[0]}${request.slots.length > 1 ? `–${addOneHour(request.slots.at(-1))}` : ""}`
          : "di jam-jam berikut";
        const dateLabel = formatDateLong(request.date);
        const reply = availableFields.length
          ? `Ini lapangan yang tersedia ${timeDescription} pada ${dateLabel}. Ketuk lapangan pilihanmu kalau ingin booking.`
          : `Maaf, belum ada lapangan yang tersedia ${timeDescription} pada ${dateLabel}. Coba tanggal atau jam lain, ya.`;
        setMessages((prev) => [...prev, {
          from: "assistant",
          text: reply,
          fields: availableFields,
          availabilityDate: request.date,
          suggestedSlots: request.slots || [],
        }]);
        return;
      }

      const result = await api.assistantChat({ message: value, history });
      setMessages((prev) => [...prev, { from: "assistant", text: result.data.reply }]);
    } catch (error) {
      setMessages((prev) => [...prev, { from: "assistant", text: error.message || "Assistant sedang tidak tersedia." }]);
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <button
        onClick={() => setOpen((v) => !v)}
        aria-label="Buka ArenaGO Assistant"
        className="fixed bottom-24 right-5 z-[120] flex h-14 w-14 items-center justify-center rounded-full bg-onyx text-canvas shadow-overlay ring-2 ring-champagne/60 transition-transform duration-150 hover:scale-105 md:bottom-7 md:right-7"
      >
        <Icon name={open ? "close" : "auto_awesome"} size={24} />
      </button>

      {open && (
        <div className="fixed bottom-40 right-5 z-[110] flex h-[70vh] max-h-[560px] w-[calc(100%-2.5rem)] max-w-[380px] flex-col overflow-hidden rounded-lg border border-border bg-canvas shadow-overlay md:bottom-24 md:right-7">
          <div className="glass flex items-center gap-3 border-b border-border px-5 py-4">
            <span className="flex h-9 w-9 items-center justify-center rounded-full bg-onyx text-canvas">
              <Icon name="auto_awesome" size={16} />
            </span>
            <div>
              <p className="text-sm font-semibold leading-none">ArenaGO Assistant</p>
              <p className="mt-1 text-xs text-ink-faint">Concierge padel Anda</p>
            </div>
          </div>

          <div className="flex-1 space-y-3 overflow-y-auto px-5 py-4">
            {messages.map((m, i) => (
              <div
                key={i}
                className={`flex ${m.from === "user" ? "justify-end" : "justify-start"}`}
              >
                <div
                  className={`${m.fields ? "w-full" : "max-w-[80%]"} rounded-lg px-4 py-2.5 text-sm ${
                    m.from === "user"
                      ? "bg-onyx text-canvas"
                      : "border border-border bg-surface-raised text-ink"
                  }`}
                >
                  <p>{m.text}</p>
                  {m.fields?.length > 0 && (
                    <div className="mt-3 space-y-2">
                      {m.fields.map(({ field, slots }) => (
                        <button
                          key={field.id}
                          type="button"
                          aria-label={`Booking ${field.name}`}
                          onClick={() => {
                            navigate(`/jadwal/${field.id}`, {
                              state: {
                                availabilitySuggestion: {
                                  date: m.availabilityDate,
                                  slots: m.suggestedSlots,
                                },
                              },
                            });
                            setOpen(false);
                          }}
                          className="flex w-full items-center gap-3 rounded-md border border-border bg-canvas p-2 text-left transition-colors hover:border-sage focus:outline-none focus:ring-2 focus:ring-sage"
                        >
                          <div className="flex h-14 w-16 shrink-0 items-center justify-center overflow-hidden rounded bg-surface-raised">
                            {field.imageUrl ? (
                              <img src={field.imageUrl} alt="" className="h-full w-full object-cover" />
                            ) : (
                              <Icon name="sports_tennis" size={24} className="text-sage" />
                            )}
                          </div>
                          <span className="min-w-0 flex-1">
                            <span className="block truncate font-semibold">{field.name}</span>
                            <span className="mt-0.5 block truncate text-xs text-ink-muted">{field.location}</span>
                            <span className="mt-1 block truncate text-xs text-sage">
                              {m.suggestedSlots.length
                                ? `${m.suggestedSlots[0]}${m.suggestedSlots.length > 1 ? `–${addOneHour(m.suggestedSlots.at(-1))}` : ""}`
                                : `Tersedia ${slots.slice(0, 4).map((slot) => slot.start_time).join(", ")}${slots.length > 4 ? ` +${slots.length - 4} slot` : ""}`}
                            </span>
                            <span className="mt-0.5 block text-xs text-ink-faint">{formatIDR(field.pricePerHour)}/jam</span>
                          </span>
                          <Icon name="chevron_right" size={20} className="shrink-0 text-ink-faint" />
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            ))}
            {loading && <div className="max-w-[80%] rounded-lg border border-border bg-surface-raised px-4 py-2.5 text-sm text-ink-faint">Sedang berpikir...</div>}
            <div ref={messagesEndRef} />
          </div>

          <div className="border-t border-border px-5 py-3">
            <div className="mb-3 flex flex-wrap gap-1.5 no-scrollbar">
              {conciergeQuickActions.map((action) => (
                <button
                  key={action}
                  onClick={() => sendMessage(action)}
                  className="rounded-full border border-border px-3 py-1.5 text-[11px] text-ink-muted hover:border-sage hover:text-ink"
                >
                  {action}
                </button>
              ))}
            </div>
            <form
              onSubmit={(e) => {
                e.preventDefault();
                sendMessage();
              }}
              className="flex items-center gap-2"
            >
              <input
                value={draft}
                onChange={(e) => setDraft(e.target.value)}
                placeholder="Tulis pesan..."
                disabled={loading}
                className="h-11 flex-1 rounded-full border border-border bg-surface-raised px-4 text-sm focus:border-onyx focus:outline-none"
              />
              <button
                type="submit"
                disabled={loading}
                aria-label="Kirim"
                className="flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-onyx text-canvas"
              >
                <Icon name="arrow_upward" size={18} />
              </button>
            </form>
          </div>
        </div>
      )}
    </>
  );
}

function formatDateLong(value) {
  const [year, month, day] = value.split("-").map(Number);
  return new Intl.DateTimeFormat("id-ID", {
    weekday: "long",
    day: "numeric",
    month: "long",
    year: "numeric",
  }).format(new Date(year, month - 1, day));
}

function addOneHour(value) {
  const hour = Number(value.slice(0, 2)) + 1;
  return `${String(hour).padStart(2, "0")}:00`;
}
