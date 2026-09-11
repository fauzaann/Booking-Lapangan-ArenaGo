import { useState } from "react";
import Icon from "../ui/Icon";
import { conciergeSeedMessages, conciergeQuickActions } from "../../lib/mockData";

export default function ConciergeWidget() {
  const [open, setOpen] = useState(false);
  const [messages, setMessages] = useState(conciergeSeedMessages);
  const [draft, setDraft] = useState("");

  function sendMessage(text) {
    const value = text ?? draft;
    if (!value.trim()) return;
    setMessages((prev) => [...prev, { from: "user", text: value }]);
    setDraft("");
  }

  return (
    <>
      <button
        onClick={() => setOpen((v) => !v)}
        aria-label="Buka Atelier Assistant"
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
              <p className="text-sm font-semibold leading-none">Atelier Assistant</p>
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
                  className={`max-w-[80%] rounded-lg px-4 py-2.5 text-sm ${
                    m.from === "user"
                      ? "bg-onyx text-canvas"
                      : "border border-border bg-surface-raised text-ink"
                  }`}
                >
                  {m.text}
                </div>
              </div>
            ))}
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
                className="h-11 flex-1 rounded-full border border-border bg-surface-raised px-4 text-sm focus:border-onyx focus:outline-none"
              />
              <button
                type="submit"
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
