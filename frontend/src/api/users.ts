import { apiFetch, API_V1 } from "./client";
import type { User } from "../types";

// Backend's UserProfileResponse omits created_at (it's immutable after
// signup, so there's no need to refetch or merge it).
export type Profile = Omit<User, "created_at">;

export function getProfile(): Promise<Profile> {
  return apiFetch<Profile>(`${API_V1}/users/profile`);
}
