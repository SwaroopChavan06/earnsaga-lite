import { apiFetch } from "./client";
import type { Offer, OfferDetail, StartResult } from "../types";

export function listOffers(search = ""): Promise<Offer[]> {
  return apiFetch<Offer[]>("/api/v1/offers", {
    params: { search },
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
