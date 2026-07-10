import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { BarChart3, TrendingUp, Users, MousePointerClick, Eye } from "lucide-react";
import { getAnalytics } from "../api/analytics";
import { Spinner } from "../components/Spinner";
import { ErrorCard } from "../components/ErrorCard";

function StatCard({
  icon: Icon,
  label,
  value,
  color,
}: {
  icon: React.ElementType;
  label: string;
  value: string | number;
  color: string;
}) {
  return (
    <div className="bg-slate-900 border border-slate-700 rounded-xl p-5">
      <div className={`inline-flex p-2 rounded-lg ${color} mb-3`}>
        <Icon className="w-4 h-4" />
      </div>
      <p className="text-2xl font-bold text-white">{value}</p>
      <p className="text-slate-400 text-xs mt-1">{label}</p>
    </div>
  );
}

export function AdminAnalytics() {
  const today = new Date().toISOString().slice(0, 10);
  const sevenDaysAgo = new Date(Date.now() - 7 * 86_400_000).toISOString().slice(0, 10);

  const [from, setFrom] = useState(sevenDaysAgo);
  const [to, setTo] = useState(today);
  const [offerId, setOfferId] = useState("");

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ["analytics", from, to, offerId],
    queryFn: () => getAnalytics({ from, to, offer_id: offerId }),
  });

  // Aggregate totals across all rows.
  const totals = (data?.by_date_offer ?? []).reduce(
    (acc, row) => ({
      impressions: acc.impressions + row.impressions,
      clicks: acc.clicks + row.clicks,
      revenue: acc.revenue + row.revenue_usd,
    }),
    { impressions: 0, clicks: 0, revenue: 0 }
  );

  const totalDAU = (data?.by_date ?? []).reduce((acc, r) => acc + r.dau, 0);
  const avgDAU =
    data && data.by_date.length > 0
      ? (totalDAU / data.by_date.length).toFixed(1)
      : "—";

  return (
    <div>
      <div className="mb-8 flex items-center gap-3">
        <BarChart3 className="w-6 h-6 text-indigo-400" />
        <div>
          <h1 className="text-2xl font-bold text-white">Analytics</h1>
          <p className="text-slate-400 text-sm mt-0.5">Admin-only platform metrics.</p>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-slate-900 border border-slate-700 rounded-xl p-5 mb-6 flex flex-wrap gap-4 items-end">
        <div>
          <label className="text-slate-400 text-xs block mb-1">From</label>
          <input
            type="date"
            value={from}
            max={to}
            onChange={(e) => setFrom(e.target.value)}
            className="bg-slate-800 border border-slate-600 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-indigo-500"
          />
        </div>
        <div>
          <label className="text-slate-400 text-xs block mb-1">To</label>
          <input
            type="date"
            value={to}
            min={from}
            onChange={(e) => setTo(e.target.value)}
            className="bg-slate-800 border border-slate-600 rounded-lg px-3 py-2 text-white text-sm focus:outline-none focus:border-indigo-500"
          />
        </div>
        <div>
          <label className="text-slate-400 text-xs block mb-1">Offer ID (optional)</label>
          <input
            type="text"
            value={offerId}
            onChange={(e) => setOfferId(e.target.value)}
            placeholder="UUID or blank for all"
            className="bg-slate-800 border border-slate-600 rounded-lg px-3 py-2 text-white text-sm placeholder-slate-500 focus:outline-none focus:border-indigo-500 w-52"
          />
        </div>
        <button
          onClick={() => refetch()}
          className="bg-indigo-600 hover:bg-indigo-500 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          Apply
        </button>
      </div>

      {isLoading && (
        <div className="flex justify-center py-20">
          <Spinner />
        </div>
      )}

      {isError && <ErrorCard message="Failed to load analytics." />}

      {data && (
        <>
          {/* Summary cards */}
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
            <StatCard
              icon={Eye}
              label="Impressions"
              value={totals.impressions.toLocaleString()}
              color="bg-blue-500/10 text-blue-400"
            />
            <StatCard
              icon={MousePointerClick}
              label="Clicks"
              value={totals.clicks.toLocaleString()}
              color="bg-purple-500/10 text-purple-400"
            />
            <StatCard
              icon={TrendingUp}
              label="Revenue"
              value={`$${totals.revenue.toFixed(2)}`}
              color="bg-green-500/10 text-green-400"
            />
            <StatCard
              icon={Users}
              label="Avg DAU"
              value={avgDAU}
              color="bg-yellow-500/10 text-yellow-400"
            />
          </div>

          {/* By date+offer table */}
          {data.by_date_offer.length > 0 && (
            <div className="bg-slate-900 border border-slate-700 rounded-xl overflow-hidden mb-6">
              <div className="px-5 py-3 border-b border-slate-700">
                <h2 className="text-white font-semibold text-sm">By Date & Offer</h2>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-slate-800">
                      <th className="text-left text-slate-400 text-xs px-5 py-3 font-medium">Date</th>
                      <th className="text-left text-slate-400 text-xs px-5 py-3 font-medium">Offer</th>
                      <th className="text-right text-slate-400 text-xs px-5 py-3 font-medium">Impressions</th>
                      <th className="text-right text-slate-400 text-xs px-5 py-3 font-medium">Clicks</th>
                      <th className="text-right text-slate-400 text-xs px-5 py-3 font-medium">Revenue</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800">
                    {data.by_date_offer.map((row, i) => (
                      <tr key={i} className="hover:bg-slate-800/50">
                        <td className="text-slate-300 px-5 py-3 whitespace-nowrap">{row.date}</td>
                        <td className="text-slate-300 px-5 py-3 max-w-[200px] truncate">
                          {row.offer_name || row.offer_id.slice(0, 8) + "…"}
                        </td>
                        <td className="text-right text-slate-300 px-5 py-3">{row.impressions}</td>
                        <td className="text-right text-slate-300 px-5 py-3">{row.clicks}</td>
                        <td className="text-right text-green-400 px-5 py-3 font-medium">
                          ${row.revenue_usd.toFixed(2)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* DAU table */}
          {data.by_date.length > 0 && (
            <div className="bg-slate-900 border border-slate-700 rounded-xl overflow-hidden">
              <div className="px-5 py-3 border-b border-slate-700">
                <h2 className="text-white font-semibold text-sm">Daily Active Users</h2>
              </div>
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-slate-800">
                    <th className="text-left text-slate-400 text-xs px-5 py-3 font-medium">Date</th>
                    <th className="text-right text-slate-400 text-xs px-5 py-3 font-medium">DAU</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800">
                  {data.by_date.map((row, i) => (
                    <tr key={i} className="hover:bg-slate-800/50">
                      <td className="text-slate-300 px-5 py-3">{row.date}</td>
                      <td className="text-right text-yellow-400 px-5 py-3 font-medium">{row.dau}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {data.by_date_offer.length === 0 && data.by_date.length === 0 && (
            <p className="text-slate-400 text-center py-12 text-sm">
              No data for the selected date range.
            </p>
          )}
        </>
      )}
    </div>
  );
}
