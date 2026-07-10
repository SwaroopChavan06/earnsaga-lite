import { Link, NavLink, useNavigate } from "react-router-dom";
import { useAuth } from "../hooks/useAuth";
import { Coins, Trophy, Wallet, BarChart3, LogOut } from "lucide-react";

export function Navbar() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate("/login");
  };

  const linkClass = ({ isActive }: { isActive: boolean }) =>
    `flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
      isActive
        ? "bg-indigo-600 text-white"
        : "text-slate-300 hover:bg-slate-700 hover:text-white"
    }`;

  return (
    <nav className="bg-slate-900 border-b border-slate-700 sticky top-0 z-50">
      <div className="max-w-6xl mx-auto px-4 h-14 flex items-center justify-between">
        <Link to="/offers" className="text-white font-bold text-lg tracking-tight flex items-center gap-2">
          <Coins className="w-5 h-5 text-yellow-400" />
          EarnSaga
        </Link>

        <div className="flex items-center gap-1">
          <NavLink to="/offers" className={linkClass}>
            <Coins className="w-4 h-4" /> Offers
          </NavLink>
          <NavLink to="/leaderboard" className={linkClass}>
            <Trophy className="w-4 h-4" /> Leaderboard
          </NavLink>
          <NavLink to="/wallet" className={linkClass}>
            <Wallet className="w-4 h-4" /> Wallet
          </NavLink>
          {user?.is_admin && (
            <NavLink to="/admin/analytics" className={linkClass}>
              <BarChart3 className="w-4 h-4" /> Analytics
            </NavLink>
          )}
        </div>

        <div className="flex items-center gap-3">
          {user && (
            <>
              <img
                src={user.avatar_url || `https://ui-avatars.com/api/?name=${encodeURIComponent(user.name)}&background=4f46e5&color=fff`}
                alt={user.name}
                className="w-7 h-7 rounded-full"
              />
              <span className="text-slate-300 text-sm hidden sm:block">{user.name}</span>
            </>
          )}
          <button
            onClick={handleLogout}
            className="flex items-center gap-1 text-slate-400 hover:text-white transition-colors text-sm"
          >
            <LogOut className="w-4 h-4" />
          </button>
        </div>
      </div>
    </nav>
  );
}
