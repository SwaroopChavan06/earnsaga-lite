// ── Auth ─────────────────────────────────────────────────────────────────────

export interface User {
  id: string;
  email: string;
  name: string;
  avatar_url: string;
  is_admin: boolean;
  created_at: string;
}

export interface AuthResponse {
  token: string;
  user: User;
}

// ── Offers ───────────────────────────────────────────────────────────────────

export interface OfferPage {
  offers: Offer[];
  total: number;
  page: number;
  limit: number;
  pages: number;
}

export interface OfferGoal {
  id: string;
  offer_id: string;
  title: string;
  instructions: string;
  reward_usd: number;
  sort_order: number;
}

export interface Offer {
  id: string;
  pubscale_id: string;
  name: string;
  description: string;
  icon_url: string;
  payout_usd: number;
  tracking_url: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface OfferDetail extends Offer {
  goals: OfferGoal[];
  status: "not_started" | "in_progress" | "completed";
}

export interface StartResult {
  status: string;
  redirect_url: string;
  already_started: boolean;
}

// ── Wallet ───────────────────────────────────────────────────────────────────

export interface WalletBalance {
  user_id: string;
  balance_usd: number;
}

export interface Transaction {
  id: string;
  user_id: string;
  amount_usd: number;
  type: string;
  offer_id?: string;
  goal_id?: string;
  offer_name?: string;
  created_at: string;
}

// ── Leaderboard ──────────────────────────────────────────────────────────────

export type LeaderboardRange = "daily" | "weekly" | "alltime";

export interface LeaderboardEntry {
  rank: number;
  user_id: string;
  name: string;
  avatar_url: string;
  coins: number;
}

// ── Analytics ────────────────────────────────────────────────────────────────

export interface DateOfferRow {
  date: string;
  offer_id: string;
  offer_name: string;
  impressions: number;
  clicks: number;
  revenue_usd: number;
}

export interface DAURow {
  date: string;
  dau: number;
}

export interface AnalyticsReport {
  by_date_offer: DateOfferRow[];
  by_date: DAURow[];
}
