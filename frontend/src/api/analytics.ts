import { apiFetch } from "./client";
import type { AnalyticsReport } from "../types";

export interface AnalyticsFilter {
  from?: string;
  to?: string;
  offer_id?: string;
}

export function getAnalytics(filter: AnalyticsFilter = {}): Promise<AnalyticsReport> {
  return apiFetch<AnalyticsReport>("/api/v1/admin/analytics", {
    params: {
      from: filter.from ?? "",
      to: filter.to ?? "",
      offer_id: filter.offer_id ?? "",
    },
  });
}
