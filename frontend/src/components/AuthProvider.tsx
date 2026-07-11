import { createContext, useState, useCallback, useEffect, ReactNode } from "react";
import type { User } from "../types";
import { getProfile } from "../api/users";

interface AuthState {
  user: User | null;
  token: string | null;
  login: (token: string, user: User) => void;
  logout: () => void;
}

export const AuthContext = createContext<AuthState | null>(null);

function loadInitialState(): { user: User | null; token: string | null } {
  try {
    const token = localStorage.getItem("token");
    const raw = localStorage.getItem("user");
    if (token && raw) return { token, user: JSON.parse(raw) as User };
  } catch {
    // corrupted storage — start fresh
  }
  return { user: null, token: null };
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState(loadInitialState);

  const login = useCallback((token: string, user: User) => {
    localStorage.setItem("token", token);
    localStorage.setItem("user", JSON.stringify(user));
    setState({ token, user });
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    setState({ token: null, user: null });
  }, []);

  // The `user` snapshot from login (including is_admin) is otherwise frozen
  // in localStorage forever. Refetch once per token so admin
  // promotions/demotions or profile edits made elsewhere (e.g. directly in
  // the DB) take effect on the next app load without forcing a re-login.
  useEffect(() => {
    if (!state.token) return;
    let cancelled = false;

    getProfile()
      .then((profile) => {
        if (cancelled) return;
        setState((prev) => {
          if (!prev.user) return prev;
          const merged = { ...prev.user, ...profile };
          localStorage.setItem("user", JSON.stringify(merged));
          return { ...prev, user: merged };
        });
      })
      .catch(() => {
        // Invalid/expired token — drop the stale session rather than leave
        // the UI stuck with credentials the API already rejected.
        if (cancelled) return;
        localStorage.removeItem("token");
        localStorage.removeItem("user");
        setState({ token: null, user: null });
      });

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- refetch only when the token itself changes (login/logout), not on every user-state update
  }, [state.token]);

  return (
    <AuthContext.Provider value={{ ...state, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}
