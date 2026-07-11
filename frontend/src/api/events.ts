import { apiFetch, API_V1 } from "./client";

type EventType = "impression" | "click";

export function trackEvent(type: EventType, offerId: string): void {
  // Fire-and-forget — analytics failures must never break the UX.
  apiFetch(`${API_V1}/events`, {
    method: "POST",
    body: JSON.stringify({
      type,
      offer_id: offerId,
      timestamp: new Date().toISOString(),
    }),
  }).catch(() => {});
}
