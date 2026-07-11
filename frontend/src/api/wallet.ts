import { apiFetch, API_V1 } from "./client";
import type { WalletBalance, Transaction } from "../types";

export function getWallet(): Promise<WalletBalance> {
  return apiFetch<WalletBalance>(`${API_V1}/users/wallet`);
}

export function getTransactions(): Promise<Transaction[]> {
  return apiFetch<Transaction[]>(`${API_V1}/users/wallet/transactions`);
}
