import { useState, useEffect, useRef } from "react";
import Icon from "./Icon";
import { api } from "../../lib/api";

export default function NotificationDropdown() {
  const [open, setOpen] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState(null);
  const [desktopPermission, setDesktopPermission] = useState(
    () => ("Notification" in window ? Notification.permission : "unsupported")
  );
  const dropdownRef = useRef(null);
  const seenNotificationIds = useRef(new Set());
  const hasLoadedNotifications = useRef(false);
  const audioContext = useRef(null);
  const toastTimeout = useRef(null);

  function playNotificationSound() {
    if (!audioContext.current || audioContext.current.state !== "running") return;

    const context = audioContext.current;
    const oscillator = context.createOscillator();
    const gain = context.createGain();
    const now = context.currentTime;

    oscillator.type = "sine";
    oscillator.frequency.setValueAtTime(880, now);
    gain.gain.setValueAtTime(0.001, now);
    gain.gain.exponentialRampToValueAtTime(0.12, now + 0.02);
    gain.gain.exponentialRampToValueAtTime(0.001, now + 0.32);
    oscillator.connect(gain);
    gain.connect(context.destination);
    oscillator.start(now);
    oscillator.stop(now + 0.33);
  }

  function showNotification(notification) {
    setToast(notification);
    clearTimeout(toastTimeout.current);
    toastTimeout.current = setTimeout(() => setToast(null), 7000);

    if (document.visibilityState === "visible") {
      playNotificationSound();
      return;
    }

    if ("Notification" in window && Notification.permission === "granted") {
      const desktopNotification = new Notification(notification.title, {
        body: notification.message,
        tag: notification.id,
      });
      desktopNotification.onclick = () => {
        window.focus();
        setOpen(true);
        desktopNotification.close();
      };
    }
  }

  async function fetchNotifications() {
    setLoading(true);
    try {
      const res = await api.notifications({ page: 1, limit: 10 });
      if (Array.isArray(res?.data)) {
        setItems(res.data);
      }
    } catch {
      // diamkan jika offline
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    let active = true;
    let polling = false;

    async function pollNotifications() {
      if (polling) return;
      polling = true;

      try {
        const [countResult, notificationResult] = await Promise.allSettled([
          api.unreadNotificationCount(),
          api.notifications({ page: 1, limit: 10 }),
        ]);

        if (!active) return;

        if (countResult.status === "fulfilled" && countResult.value?.data?.unread_count !== undefined) {
          setUnreadCount(countResult.value.data.unread_count);
        }

        if (notificationResult.status === "fulfilled" && Array.isArray(notificationResult.value?.data)) {
          const latestNotifications = notificationResult.value.data;
          const currentIds = new Set(latestNotifications.map((notification) => notification.id));
          setItems(latestNotifications);

          if (hasLoadedNotifications.current) {
            const newNotification = latestNotifications.find(
              (notification) => !seenNotificationIds.current.has(notification.id)
            );
            if (newNotification) showNotification(newNotification);
          } else {
            hasLoadedNotifications.current = true;
          }

          seenNotificationIds.current = currentIds;
        }
      } finally {
        polling = false;
      }
    }

    function unlockNotificationSound() {
      if (audioContext.current) return;
      const AudioContextClass = window.AudioContext || window.webkitAudioContext;
      if (!AudioContextClass) return;
      audioContext.current = new AudioContextClass();
      if (audioContext.current.state === "suspended") {
        audioContext.current.resume();
      }
    }

    document.addEventListener("pointerdown", unlockNotificationSound, { once: true });
    document.addEventListener("keydown", unlockNotificationSound, { once: true });
    document.addEventListener("visibilitychange", pollNotifications);
    window.addEventListener("focus", pollNotifications);
    const interval = setInterval(pollNotifications, 5000);
    pollNotifications();

    return () => {
      active = false;
      clearInterval(interval);
      clearTimeout(toastTimeout.current);
      document.removeEventListener("pointerdown", unlockNotificationSound);
      document.removeEventListener("keydown", unlockNotificationSound);
      document.removeEventListener("visibilitychange", pollNotifications);
      window.removeEventListener("focus", pollNotifications);
      audioContext.current?.close();
    };
  }, []);

  // Tutup dropdown saat klik di luar
  useEffect(() => {
    function handleClickOutside(event) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setOpen(false);
      }
    }
    if (open) {
      document.addEventListener("mousedown", handleClickOutside);
    }
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [open]);

  async function handleMarkRead(id, e) {
    e?.stopPropagation();
    try {
      await api.markNotificationRead(id);
      setItems((prev) =>
        prev.map((item) => (item.id === id ? { ...item, is_read: true } : item))
      );
      setUnreadCount((prev) => Math.max(0, prev - 1));
    } catch {
      // ignore
    }
  }

  async function handleMarkAllRead() {
    try {
      await api.markAllNotificationsRead();
      setItems((prev) => prev.map((item) => ({ ...item, is_read: true })));
      setUnreadCount(0);
    } catch {
      // ignore
    }
  }

  async function handleEnableDesktopNotifications() {
    const permission = await Notification.requestPermission();
    setDesktopPermission(permission);
  }

  function formatTime(isoString) {
    if (!isoString) return "";
    const date = new Date(isoString);
    return date.toLocaleDateString("id-ID", {
      day: "numeric",
      month: "short",
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  return (
    <div className="relative" ref={dropdownRef}>
      <button
        onClick={() => {
          const nextOpen = !open;
          setOpen(nextOpen);
          if (nextOpen) fetchNotifications();
        }}
        className="relative flex h-9 w-9 items-center justify-center rounded-full border border-border text-ink-muted transition-colors hover:border-ink hover:text-ink focus:outline-none"
        title="Notifikasi Pengingat"
        aria-label="Notifikasi Pengingat"
      >
        <Icon name="notifications" size={20} />
        {unreadCount > 0 && (
          <span className="absolute -top-1 -right-1 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white shadow-sm">
            {unreadCount > 9 ? "9+" : unreadCount}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute right-0 z-50 mt-2 w-80 sm:w-96 overflow-hidden rounded-xl border border-border bg-surface-raised shadow-xl">
          {/* Header */}
          <div className="flex items-center justify-between border-b border-border px-4 py-3 bg-surface">
            <div className="flex items-center gap-2">
              <span className="font-semibold text-sm text-ink">Notifikasi</span>
              {unreadCount > 0 && (
                <span className="rounded-full bg-onyx/10 px-2 py-0.5 text-xs font-medium text-onyx">
                  {unreadCount} baru
                </span>
              )}
            </div>
            <div className="flex items-center gap-3">
              {"Notification" in window && desktopPermission === "default" && (
                <button
                  onClick={handleEnableDesktopNotifications}
                  className="text-xs font-medium text-ink-muted hover:text-ink transition-colors"
                >
                  Aktifkan desktop
                </button>
              )}
              {unreadCount > 0 && (
                <button
                  onClick={handleMarkAllRead}
                  className="text-xs font-medium text-ink-muted hover:text-ink transition-colors"
                >
                  Tandai semua dibaca
                </button>
              )}
            </div>
          </div>

          {/* List Content */}
          <div className="max-h-80 overflow-y-auto divide-y divide-border">
            {loading ? (
              <div className="p-6 text-center text-xs text-ink-muted">
                Memuat notifikasi...
              </div>
            ) : items.length === 0 ? (
              <div className="p-8 text-center text-ink-muted">
                <Icon name="notifications_off" size={28} className="mx-auto mb-2 text-ink-faint" />
                <p className="text-xs">Belum ada notifikasi jadwal bermain.</p>
              </div>
            ) : (
              items.map((n) => (
                <div
                  key={n.id}
                  onClick={(e) => !n.is_read && handleMarkRead(n.id, e)}
                  className={`p-3.5 transition-colors cursor-pointer ${
                    !n.is_read ? "bg-onyx/5 hover:bg-onyx/10" : "hover:bg-surface"
                  }`}
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-center gap-1.5">
                      <span className="text-sm font-semibold text-ink line-clamp-1">
                        {n.title}
                      </span>
                    </div>
                    {!n.is_read && (
                      <span className="mt-1 h-2 w-2 shrink-0 rounded-full bg-blue-500" />
                    )}
                  </div>
                  <p className="mt-1 text-xs text-ink-muted leading-relaxed line-clamp-2">
                    {n.message}
                  </p>
                  <span className="mt-2 block text-[10px] text-ink-faint">
                    {formatTime(n.created_at)}
                  </span>
                </div>
              ))
            )}
          </div>
        </div>
      )}

      {toast && (
        <div
          role="status"
          aria-live="polite"
          className="fixed right-4 top-20 z-[60] w-[min(24rem,calc(100vw-2rem))] rounded-xl border border-border bg-surface-raised p-4 shadow-xl"
        >
          <div className="flex items-start gap-3">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-onyx/5 text-ink">
              <Icon name="notifications_active" size={20} />
            </span>
            <button
              onClick={() => setToast(null)}
              className="ml-auto rounded-full p-1 text-ink-faint hover:bg-surface hover:text-ink"
              aria-label="Tutup notifikasi"
            >
              <Icon name="close" size={18} />
            </button>
          </div>
          <p className="mt-3 text-sm font-semibold text-ink">{toast.title}</p>
          <p className="mt-1 text-xs leading-relaxed text-ink-muted">{toast.message}</p>
        </div>
      )}
    </div>
  );
}
