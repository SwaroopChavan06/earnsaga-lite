import { useState, useEffect } from "react";
import { Trophy, Medal, Crown } from "lucide-react";
import { useSSE } from "../hooks/useSSE";
import { getLeaderboard, streamLeaderboardUrl } from "../api/leaderboard";
import { useQuery } from "@tanstack/react-query";
import { Spinner } from "../components/Spinner";
import type { LeaderboardEntry, LeaderboardRange } from "../types";

const TABS: { id: LeaderboardRange; label: string }[] = [
  { id: "daily", label: "Daily" },
  { id: "weekly", label: "Weekly" },
  { id: "alltime", label: "All Time" },
];

function RankBadge({ rank }: { rank: number }) {
  if (rank === 1) return <Crown className="w-5 h-5 text-yellow-400" />;
  if (rank === 2) return <Medal className="w-5 h-5 text-slate-300" />;
  if (rank === 3) return <Medal className="w-5 h-5 text-amber-600" />;
  return <span className="text-slate-400 text-sm w-5 text-center">{rank}</span>;
}

function LeaderboardTable({ entries }: { entries: LeaderboardEntry[] }) {
  if (entries.length === 0) {
    return (
      <p className="text-slate-400 text-center py-12 text-sm">
        No rankings yet for this period.
      </p>
    );
  }

  return (
    <ul className="divide-y divide-slate-800">
      {entries.map((entry) => (
        <li
          key={entry.user_id}
          className={`px-5 py-3.5 flex items-center gap-4 ${
            entry.rank <= 3 ? "bg-slate-800/50" : ""
          }`}
        >
          <div className="w-6 flex justify-center shrink-0">
            <RankBadge rank={entry.rank} />
          </div>
          <img
            src={
              entry.avatar_url ||
              `https://ui-avatars.com/api/?name=${encodeURIComponent(entry.name || "?")}&background=1e1b4b&color=a5b4fc&size=64`
            }
            alt={entry.name}
            className="w-8 h-8 rounded-full shrink-0"
          />
          <span className="flex-1 text-white text-sm font-medium truncate">
            {entry.name || "Anonymous"}
          </span>
          <span className="text-yellow-400 font-semibold text-sm">
            ${entry.coins.toFixed(2)}
          </span>
        </li>
      ))}
    </ul>
  );
}

export function Leaderboard() {
  const [range, setRange] = useState<LeaderboardRange>("daily");
  const [sseEntries, setSseEntries] = useState<LeaderboardEntry[] | null>(null);
  const [sseUrl, setSseUrl] = useState("");

  // Build the SSE URL only on the client (needs localStorage token).
  useEffect(() => {
    setSseUrl(streamLeaderboardUrl(range));
    setSseEntries(null); // reset when range changes so we show the REST snapshot first
  }, [range]);

  // SSE stream — when it delivers data, use it instead of the REST snapshot.
  const { data: streamData, error: sseError } = useSSE<LeaderboardEntry[]>(sseUrl);

  useEffect(() => {
    if (streamData) setSseEntries(streamData);
  }, [streamData]);

  // REST snapshot — used as initial data before the first SSE frame, and as
  // fallback when SSE errors out.
  const { data: restData, isLoading } = useQuery({
    queryKey: ["leaderboard", range],
    queryFn: () => getLeaderboard(range),
    enabled: !sseEntries, // stop polling once SSE is delivering
  });

  const entries = sseEntries ?? restData ?? [];

  return (
    <div className="max-w-2xl">
      <div className="mb-8 flex items-center gap-3">
        <Trophy className="w-6 h-6 text-yellow-400" />
        <div>
          <h1 className="text-2xl font-bold text-white">Leaderboard</h1>
          <p className="text-slate-400 text-sm mt-0.5">
            Top 50 earners{" "}
            {!sseError && sseEntries && (
              <span className="text-green-400 text-xs">● live</span>
            )}
          </p>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 bg-slate-800 rounded-lg p-1 mb-6 w-fit">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setRange(tab.id)}
            className={`px-4 py-1.5 rounded-md text-sm font-medium transition-colors ${
              range === tab.id
                ? "bg-indigo-600 text-white"
                : "text-slate-400 hover:text-white"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Table */}
      <div className="bg-slate-900 border border-slate-700 rounded-2xl overflow-hidden">
        <div className="px-5 py-3 border-b border-slate-700 flex items-center justify-between">
          <span className="text-slate-400 text-xs uppercase tracking-wider">Rank</span>
          <span className="text-slate-400 text-xs uppercase tracking-wider">Earned</span>
        </div>
        {isLoading && !sseEntries ? (
          <div className="flex justify-center py-12">
            <Spinner />
          </div>
        ) : (
          <LeaderboardTable entries={entries} />
        )}
      </div>
    </div>
  );
}
