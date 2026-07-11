import { useState, useEffect, useCallback, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { Search, ExternalLink, ChevronLeft, ChevronRight } from "lucide-react";
import { listOffers } from "../api/offers";
import { trackEvent } from "../api/events";
import { Spinner } from "../components/Spinner";
import { ErrorCard } from "../components/ErrorCard";
import { flattenText } from "../utils/text";
import type { Offer } from "../types";

const PAGE_LIMIT = 20;

function OfferCard({ offer }: { offer: Offer }) {
  const navigate = useNavigate();
  const ref = useRef<HTMLButtonElement>(null);

  // Fire impression only when the card enters the viewport (≥50% visible),
  // not on every mount — avoids spamming events for off-screen cards.
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          trackEvent("impression", offer.id);
          observer.disconnect(); // fire once per card lifetime
        }
      },
      { threshold: 0.5 }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [offer.id]);

  return (
    <button
      ref={ref}
      onClick={() => navigate(`/offers/${offer.id}`)}
      className="group bg-slate-900 border border-slate-700 rounded-xl p-5 text-left hover:border-indigo-500 hover:bg-slate-800 transition-all"
    >
      <div className="flex items-start gap-4">
        <img
          src={offer.icon_url || `https://ui-avatars.com/api/?name=${encodeURIComponent(offer.name)}&background=1e1b4b&color=a5b4fc&size=80`}
          alt={offer.name}
          className="w-14 h-14 rounded-xl object-cover shrink-0"
          onError={(e) => {
            (e.target as HTMLImageElement).src = `https://ui-avatars.com/api/?name=${encodeURIComponent(offer.name)}&background=1e1b4b&color=a5b4fc&size=80`;
          }}
        />
        <div className="flex-1 min-w-0">
          <h3 className="text-white font-semibold text-sm group-hover:text-indigo-300 transition-colors truncate">
            {offer.name}
          </h3>
          <p className="text-slate-400 text-xs mt-1 line-clamp-2">{flattenText(offer.description)}</p>
        </div>
      </div>
      <div className="mt-4 flex items-center justify-between">
        <span className="text-yellow-400 font-bold text-sm">
          ${(offer.total_payout ?? 0).toFixed(2)}
        </span>
        <span className="flex items-center gap-1 text-indigo-400 text-xs group-hover:text-indigo-300">
          View details <ExternalLink className="w-3 h-3" />
        </span>
      </div>
    </button>
  );
}

export function Offers() {
  const [search, setSearch] = useState("");
  const [debouncedSearch, setDebouncedSearch] = useState("");
  const [page, setPage] = useState(1);

  // Debounce: only hit the backend 400ms after typing stops.
  const debounce = useCallback((value: string) => {
    const t = setTimeout(() => {
      setDebouncedSearch(value);
      setPage(1); // reset to page 1 on new search
    }, 400);
    return () => clearTimeout(t);
  }, []);

  useEffect(() => {
    return debounce(search);
  }, [search, debounce]);

  const { data, isLoading, isError } = useQuery({
    queryKey: ["offers", debouncedSearch, page],
    queryFn: () => listOffers({ search: debouncedSearch, page, limit: PAGE_LIMIT }),
  });

  const offers = data?.offers ?? [];
  const totalPages = data?.pages ?? 1;
  const total = data?.total ?? 0;

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-white">Offers</h1>
        <p className="text-slate-400 text-sm mt-1">Complete offers to earn rewards.</p>
      </div>

      {/* Search */}
      <div className="relative mb-6 max-w-md">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search offers…"
          className="w-full bg-slate-800 border border-slate-600 rounded-lg pl-9 pr-4 py-2.5 text-white text-sm placeholder-slate-400 focus:outline-none focus:border-indigo-500"
        />
      </div>

      {isLoading && (
        <div className="flex justify-center py-20">
          <Spinner />
        </div>
      )}

      {isError && <ErrorCard message="Failed to load offers. Is the backend running?" />}

      {!isLoading && !isError && offers.length === 0 && (
        <p className="text-slate-400 text-center py-20">
          {debouncedSearch ? `No offers found for "${debouncedSearch}"` : "No offers available."}
        </p>
      )}

      {offers.length > 0 && (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {offers.map((offer) => (
              <OfferCard key={offer.id} offer={offer} />
            ))}
          </div>

          {/* Pagination */}
          <div className="mt-8 flex items-center justify-between">
            <p className="text-slate-400 text-sm">
              Showing {(page - 1) * PAGE_LIMIT + 1}–{Math.min(page * PAGE_LIMIT, total)} of {total} offers
            </p>
            <div className="flex items-center gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="flex items-center gap-1 px-3 py-2 rounded-lg bg-slate-800 border border-slate-700 text-slate-300 hover:bg-slate-700 disabled:opacity-40 disabled:cursor-not-allowed text-sm transition-colors"
              >
                <ChevronLeft className="w-4 h-4" /> Prev
              </button>

              <span className="text-slate-400 text-sm px-2">
                Page {page} of {totalPages}
              </span>

              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
                className="flex items-center gap-1 px-3 py-2 rounded-lg bg-slate-800 border border-slate-700 text-slate-300 hover:bg-slate-700 disabled:opacity-40 disabled:cursor-not-allowed text-sm transition-colors"
              >
                Next <ChevronRight className="w-4 h-4" />
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
