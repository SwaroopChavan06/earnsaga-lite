import { Routes, Route, Navigate } from "react-router-dom";
import { ProtectedRoute } from "./components/ProtectedRoute";
import { Navbar } from "./components/Navbar";
import { Login } from "./pages/Login";
import { Offers } from "./pages/Offers";
import { OfferDetail } from "./pages/OfferDetail";
import { Wallet } from "./pages/Wallet";
import { Leaderboard } from "./pages/Leaderboard";
import { AdminAnalytics } from "./pages/AdminAnalytics";
import { useAuth } from "./hooks/useAuth";

function Layout() {
  return (
    <div className="min-h-screen bg-slate-950 text-white">
      <Navbar />
      <main className="max-w-6xl mx-auto px-4 py-8">
        <Routes>
          <Route element={<ProtectedRoute />}>
            <Route path="/offers" element={<Offers />} />
            <Route path="/offers/:id" element={<OfferDetail />} />
            <Route path="/wallet" element={<Wallet />} />
            <Route path="/leaderboard" element={<Leaderboard />} />
          </Route>
          <Route element={<ProtectedRoute adminOnly />}>
            <Route path="/admin/analytics" element={<AdminAnalytics />} />
          </Route>
        </Routes>
      </main>
    </div>
  );
}

export default function App() {
  const { token } = useAuth();
  return (
    <Routes>
      <Route path="/login" element={token ? <Navigate to="/offers" replace /> : <Login />} />
      <Route path="/*" element={<Layout />} />
      <Route path="/" element={<Navigate to="/offers" replace />} />
    </Routes>
  );
}
