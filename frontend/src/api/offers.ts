import { apiFetch } from "./client";
import type { OfferPage, OfferDetail, StartResult } from "../types";

export interface ListOffersParams {
  search?: string;
  page?: number;
  limit?: number;
}

export function listOffers({ search = "", page = 1, limit = 20 }: ListOffersParams = {}): Promise<OfferPage> {
  return apiFetch<OfferPage>("/api/v1/offers", {
    params: {
      search,
      page: String(page),
      limit: String(limit),
    },
  });
}

export function getOfferDetail(id: string): Promise<OfferDetail> {
  return apiFetch<OfferDetail>(`/api/v1/offers/${id}`);
}

export function startOffer(id: string): Promise<StartResult> {
  return apiFetch<StartResult>(`/api/v1/offers/${id}/start`, {
    method: "POST",
  });
}
