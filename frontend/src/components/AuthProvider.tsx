import { createContext, useState, useCallback, ReactNode } from "react";
import type { User } from "../types";

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

  return (
    <AuthContext.Provider value={{ ...state, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}
