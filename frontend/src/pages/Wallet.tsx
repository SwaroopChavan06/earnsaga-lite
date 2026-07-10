import { useQuery } from "@tanstack/react-query";
import { Wallet as WalletIcon, ArrowDownCircle } from "lucide-react";
import { getWallet, getTransactions } from "../api/wallet";
import { Spinner } from "../components/Spinner";
import { ErrorCard } from "../components/ErrorCard";
import { format } from "date-fns";

export function Wallet() {
  const {
    data: balance,
    isLoading: balLoading,
    isError: balError,
  } = useQuery({
    queryKey: ["wallet"],
    queryFn: getWallet,
  });

  const {
    data: transactions,
    isLoading: txLoading,
    isError: txError,
  } = useQuery({
    queryKey: ["transactions"],
    queryFn: getTransactions,
  });

  const isLoading = balLoading || txLoading;
  const isError = balError || txError;

  if (isLoading) {
    return (
      <div className="flex justify-center py-20">
        <Spinner />
      </div>
    );
  }

  if (isError) {
    return <ErrorCard message="Failed to load wallet data." />;
  }

  return (
    <div className="max-w-2xl">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-white">Wallet</h1>
        <p className="text-slate-400 text-sm mt-1">Your earnings and transaction history.</p>
      </div>

      {/* Balance card */}
      <div className="bg-gradient-to-br from-indigo-900 to-slate-900 border border-indigo-700/50 rounded-2xl p-6 mb-8">
        <div className="flex items-center gap-3 mb-2">
          <WalletIcon className="w-5 h-5 text-indigo-300" />
          <span className="text-indigo-300 text-sm font-medium">Total Balance</span>
        </div>
        <p className="text-4xl font-bold text-white">
          ${(balance?.balance_usd ?? 0).toFixed(2)}
        </p>
        <p className="text-slate-400 text-xs mt-2">Earned from completed offers</p>
      </div>

      {/* Transactions */}
      <div className="bg-slate-900 border border-slate-700 rounded-2xl overflow-hidden">
        <div className="px-5 py-4 border-b border-slate-700">
          <h2 className="text-white font-semibold">Transaction History</h2>
        </div>

        {!transactions || transactions.length === 0 ? (
          <p className="text-slate-400 text-center py-12 text-sm">
            No transactions yet. Complete an offer to earn rewards!
          </p>
        ) : (
          <ul className="divide-y divide-slate-800">
            {transactions.map((tx) => (
              <li key={tx.id} className="px-5 py-4 flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="bg-green-500/10 border border-green-500/20 rounded-full p-2">
                    <ArrowDownCircle className="w-4 h-4 text-green-400" />
                  </div>
                  <div>
                    <p className="text-white text-sm font-medium">
                      {tx.offer_name ?? "Reward"}
                    </p>
                    <p className="text-slate-400 text-xs mt-0.5">
                      {format(new Date(tx.created_at), "MMM d, yyyy · h:mm a")}
                    </p>
                  </div>
                </div>
                <span className="text-green-400 font-semibold text-sm">
                  +${tx.amount_usd.toFixed(2)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
