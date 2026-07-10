import { apiFetch } from "./client";
import type { LeaderboardEntry, LeaderboardRange } from "../types";

export function getLeaderboard(range: LeaderboardRange): Promise<LeaderboardEntry[]> {
  return apiFetch<LeaderboardEntry[]>("/api/v1/leaderboard", {
    params: { range },
  });
}

export function streamLeaderboardUrl(range: LeaderboardRange): string {
  const base = import.meta.env.VITE_API_URL ?? "";
  const token = localStorage.getItem("token") ?? "";
  return `${base}/api/v1/leaderboard/stream?range=${range}&token=${token}`;
}
