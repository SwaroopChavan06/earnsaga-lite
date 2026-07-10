import { apiFetch } from "./client";
import type { WalletBalance, Transaction } from "../types";

export function getWallet(): Promise<WalletBalance> {
  return apiFetch<WalletBalance>("/api/v1/users/wallet");
}

export function getTransactions(): Promise<Transaction[]> {
  return apiFetch<Transaction[]>("/api/v1/users/wallet/transactions");
}
