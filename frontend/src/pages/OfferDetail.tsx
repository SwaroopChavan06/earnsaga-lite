import { useParams, useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, CheckCircle, Clock, PlayCircle, Award } from "lucide-react";
import { getOfferDetail, startOffer } from "../api/offers";
import { trackEvent } from "../api/events";
import { Spinner } from "../components/Spinner";
import { ErrorCard } from "../components/ErrorCard";

const statusConfig = {
  not_started: {
    label: "Start Offer",
    icon: PlayCircle,
    className: "bg-indigo-600 hover:bg-indigo-500 text-white",
  },
  in_progress: {
    label: "In Progress",
    icon: Clock,
    className: "bg-yellow-600 hover:bg-yellow-500 text-white",
  },
  completed: {
    label: "Completed",
    icon: CheckCircle,
    className: "bg-green-700 text-white cursor-default",
  },
};

export function OfferDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: offer, isLoading, isError } = useQuery({
    queryKey: ["offer", id],
    queryFn: () => getOfferDetail(id!),
    enabled: !!id,
  });

  const startMutation = useMutation({
    mutationFn: () => {
      trackEvent("click", id!);
      return startOffer(id!);
    },
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: ["offer", id] });
      // Redirect to advertiser tracking URL
      if (result.redirect_url) {
        window.open(result.redirect_url, "_blank", "noopener,noreferrer");
      }
    },
  });

  if (isLoading) {
    return (
      <div className="flex justify-center py-20">
        <Spinner />
      </div>
    );
  }

  if (isError || !offer) {
    return <ErrorCard message="Failed to load offer details." />;
  }

  const cfg = statusConfig[offer.status];
  const Icon = cfg.icon;
  const isCompleted = offer.status === "completed";

  return (
    <div className="max-w-2xl">
      <button
        onClick={() => navigate(-1)}
        className="flex items-center gap-2 text-slate-400 hover:text-white text-sm mb-6 transition-colors"
      >
        <ArrowLeft className="w-4 h-4" /> Back to Offers
      </button>

      {/* Header */}
      <div className="bg-slate-900 border border-slate-700 rounded-2xl p-6 mb-6">
        <div className="flex items-start gap-5">
          <img
            src={offer.icon_url || `https://ui-avatars.com/api/?name=${encodeURIComponent(offer.name)}&background=1e1b4b&color=a5b4fc&size=120`}
            alt={offer.name}
            className="w-20 h-20 rounded-xl object-cover shrink-0"
            onError={(e) => {
              (e.target as HTMLImageElement).src = `https://ui-avatars.com/api/?name=${encodeURIComponent(offer.name)}&background=1e1b4b&color=a5b4fc&size=120`;
            }}
          />
          <div className="flex-1">
            <h1 className="text-xl font-bold text-white">{offer.name}</h1>
            <p className="text-slate-400 text-sm mt-2">{offer.description}</p>
            <div className="mt-3 flex items-center gap-3">
              <span className="bg-yellow-500/10 text-yellow-400 border border-yellow-500/30 rounded-full px-3 py-1 text-sm font-semibold">
                ${offer.payout_usd.toFixed(2)} total payout
              </span>
            </div>
          </div>
        </div>

        {/* CTA */}
        <div className="mt-6">
          <button
            onClick={() => !isCompleted && startMutation.mutate()}
            disabled={startMutation.isPending || isCompleted}
            className={`w-full flex items-center justify-center gap-2 py-3 px-6 rounded-xl font-semibold transition-colors ${cfg.className} disabled:opacity-70`}
          >
            <Icon className="w-5 h-5" />
            {startMutation.isPending ? "Starting…" : cfg.label}
          </button>
          {startMutation.isError && (
            <p className="text-red-400 text-xs mt-2 text-center">
              {startMutation.error instanceof Error ? startMutation.error.message : "Something went wrong"}
            </p>
          )}
          {offer.status === "in_progress" && (
            <p className="text-slate-400 text-xs mt-2 text-center">
              You've already started this offer. Clicking will reopen the tracking link.
            </p>
          )}
        </div>
      </div>

      {/* Goals */}
      {offer.goals && offer.goals.length > 0 && (
        <div className="bg-slate-900 border border-slate-700 rounded-2xl p-6">
          <h2 className="text-white font-semibold mb-4 flex items-center gap-2">
            <Award className="w-5 h-5 text-indigo-400" />
            Goals & Steps
          </h2>
          <div className="space-y-3">
            {offer.goals.map((goal, i) => (
              <div
                key={goal.id}
                className="bg-slate-800 border border-slate-700 rounded-xl p-4"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="flex items-start gap-3">
                    <span className="bg-indigo-600/20 text-indigo-400 border border-indigo-500/30 rounded-full w-6 h-6 flex items-center justify-center text-xs font-bold shrink-0 mt-0.5">
                      {i + 1}
                    </span>
                    <div>
                      <p className="text-white text-sm font-medium">{goal.title}</p>
                      <p className="text-slate-400 text-xs mt-1">{goal.instructions}</p>
                    </div>
                  </div>
                  {goal.reward_usd > 0 && (
                    <span className="text-yellow-400 text-sm font-semibold shrink-0">
                      ${goal.reward_usd.toFixed(2)}
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
