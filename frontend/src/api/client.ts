const BASE = import.meta.env.VITE_API_URL ?? "";

// Version constants — import the one you need in each domain file.
// When a single endpoint has a breaking change, only that file's import
// changes from API_V1 to API_V2; all other domains are untouched.
export const API_V1 = "/api/v1";
// export const API_V2 = "/api/v2"; // uncomment when a v2 route ships

function getToken(): string | null {
  return localStorage.getItem("token");
}

interface RequestOptions extends RequestInit {
  params?: Record<string, string>;
}

export async function apiFetch<T>(
  path: string,
  options: RequestOptions = {}
): Promise<T> {
  const { params, ...init } = options;

  // path must already include the version prefix, e.g. /api/v1/offers
  let url = `${BASE}${path}`;
  if (params) {
    const qs = new URLSearchParams(
      Object.fromEntries(Object.entries(params).filter(([, v]) => v !== ""))
    ).toString();
    if (qs) url += `?${qs}`;
  }

  const token = getToken();
  const headers: HeadersInit = {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...init.headers,
  };

  const res = await fetch(url, { ...init, headers });
  if (!res.ok) {
    const body = await res.text();
    throw new Error(body || `HTTP ${res.status}`);
  }
  return res.json() as Promise<T>;
}
