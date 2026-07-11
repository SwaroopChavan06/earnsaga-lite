import { apiFetch, API_V1 } from "./client";

type EventType = "impression" | "click";

interface QueuedEvent {
  type: EventType;
  offer_id: string;
  timestamp: string;
}

// Client-side batching: impressions fire per offer card (up to 20+ per page
// view), so sending one HTTP request per event doesn't scale. Instead we
// buffer events in memory and flush them as a single request, whichever
// trigger fires first:
//   - FLUSH_INTERVAL_MS elapses since the last flush
//   - the queue reaches FLUSH_THRESHOLD events
//   - the tab is hidden/closed (flushed via sendBeacon for guaranteed delivery)
const FLUSH_INTERVAL_MS = 2000;
const FLUSH_THRESHOLD = 15;

let queue: QueuedEvent[] = [];

function drain(): QueuedEvent[] {
  const batch = queue;
  queue = [];
  return batch;
}

function flush(useBeacon = false): void {
  if (queue.length === 0) return;
  const batch = drain();
  const body = JSON.stringify({ events: batch });

  if (useBeacon && navigator.sendBeacon) {
    // sendBeacon can't set an Authorization header, so the JWT travels as
    // a query param instead — the backend auth middleware already accepts
    // this fallback for the same reason SSE (EventSource) needs it.
    const base = import.meta.env.VITE_API_URL ?? "";
    const token = localStorage.getItem("token") ?? "";
    const url = `${base}${API_V1}/events/batch?token=${encodeURIComponent(token)}`;
    const sent = navigator.sendBeacon(url, new Blob([body], { type: "application/json" }));
    if (sent) return;
    // fall through to fetch if the beacon couldn't be queued (rare)
  }

  apiFetch(`${API_V1}/events/batch`, {
    method: "POST",
    body,
  }).catch(() => {}); // analytics failures must never break the UX
}

if (typeof window !== "undefined") {
  setInterval(() => flush(false), FLUSH_INTERVAL_MS);

  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "hidden") flush(true);
  });
  window.addEventListener("pagehide", () => flush(true));
}

// trackEvent's signature is unchanged from the pre-batching version — no
// caller (Offers.tsx, OfferDetail.tsx) needs to know batching happens.
export function trackEvent(type: EventType, offerId: string): void {
  queue.push({ type, offer_id: offerId, timestamp: new Date().toISOString() });
  if (queue.length >= FLUSH_THRESHOLD) flush(false);
}
