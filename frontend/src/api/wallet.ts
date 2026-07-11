import { apiFetch, API_V1 } from "./client";
import type { WalletBalance, Transaction, WalletSummary } from "../types";

export function getWallet(): Promise<WalletBalance> {
  return apiFetch<WalletBalance>(`${API_V1}/users/wallet`);
}

export function getTransactions(): Promise<Transaction[]> {
  return apiFetch<Transaction[]>(`${API_V1}/users/wallet/transactions`);
}

// Collocated endpoint — balance + transactions in one request instead of
// two, since the Wallet page always needs both together.
export function getWalletSummary(): Promise<WalletSummary> {
  return apiFetch<WalletSummary>(`${API_V1}/users/wallet/summary`);
}
