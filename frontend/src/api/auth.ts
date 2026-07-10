import { apiFetch } from "./client";
import type { AuthResponse } from "../types";

export function googleLogin(idToken: string): Promise<AuthResponse> {
  return apiFetch<AuthResponse>("/api/v1/auth/google", {
    method: "POST",
    body: JSON.stringify({ id_token: idToken }),
  });
}
