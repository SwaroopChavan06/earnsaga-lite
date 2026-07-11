import { apiFetch, API_VERSION } from "./client";
import type { LeaderboardEntry, LeaderboardRange } from "../types";

export function getLeaderboard(range: LeaderboardRange): Promise<LeaderboardEntry[]> {
  return apiFetch<LeaderboardEntry[]>("/leaderboard", {
    params: { range },
  });
}

export function streamLeaderboardUrl(range: LeaderboardRange): string {
  const base = import.meta.env.VITE_API_URL ?? "";
  const token = localStorage.getItem("token") ?? "";
  return `${base}${API_VERSION}/leaderboard/stream?range=${range}&token=${token}`;
}
