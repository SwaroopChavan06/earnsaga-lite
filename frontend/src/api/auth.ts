import { apiFetch } from "./client";
import type { AuthResponse } from "../types";

export function googleLogin(idToken: string): Promise<AuthResponse> {
  return apiFetch<AuthResponse>("/auth/google", {
    method: "POST",
    body: JSON.stringify({ id_token: idToken }),
  });
}
