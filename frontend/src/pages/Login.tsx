import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { GoogleLogin, type CredentialResponse } from "@react-oauth/google";
import { useAuth } from "../hooks/useAuth";
import { googleLogin } from "../api/auth";
import { Coins } from "lucide-react";

export function Login() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleSuccess = async (cred: CredentialResponse) => {
    if (!cred.credential) return;
    setLoading(true);
    setError(null);
    try {
      const resp = await googleLogin(cred.credential);
      login(resp.token, resp.user);
      navigate("/offers", { replace: true });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Login failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        {/* Brand */}
        <div className="text-center mb-10">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-indigo-600 mb-4">
            <Coins className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-3xl font-bold text-white">EarnSaga</h1>
          <p className="text-slate-400 mt-2 text-sm">Complete offers. Earn rewards.</p>
        </div>

        {/* Card */}
        <div className="bg-slate-900 border border-slate-700 rounded-2xl p-8 shadow-xl">
          <h2 className="text-white font-semibold text-lg mb-6 text-center">
            Sign in to continue
          </h2>

          {error && (
            <p className="text-red-400 text-sm text-center mb-4 bg-red-950 border border-red-800 rounded-lg p-3">
              {error}
            </p>
          )}

          <div className="flex justify-center">
            {loading ? (
              <div className="h-10 flex items-center text-slate-400 text-sm">Signing in…</div>
            ) : (
              <GoogleLogin
                onSuccess={handleSuccess}
                onError={() => setError("Google sign-in was cancelled or failed")}
                theme="filled_black"
                shape="rectangular"
                size="large"
                text="signin_with"
                useOneTap
              />
            )}
          </div>

          <p className="text-slate-500 text-xs text-center mt-6">
            By signing in you agree to use this platform for testing purposes only.
          </p>
        </div>
      </div>
    </div>
  );
}
