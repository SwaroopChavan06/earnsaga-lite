import { apiFetch } from "./client";
import type { WalletBalance, Transaction } from "../types";

export function getWallet(): Promise<WalletBalance> {
  return apiFetch<WalletBalance>("/users/wallet");
}

export function getTransactions(): Promise<Transaction[]> {
  return apiFetch<Transaction[]>("/users/wallet/transactions");
}
