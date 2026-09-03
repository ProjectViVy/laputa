export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function send<T>(path: string, init: RequestInit): Promise<T> {
  const token = sessionStorage.getItem("garden.capability_token")?.trim();
  const auth: Record<string, string> = token ? { Authorization: `Bearer ${token}` } : {};
  const res = await fetch(path, {
    ...init,
    headers: { Accept: "application/json", ...auth, ...(init.headers ?? {}) },
  });
  if (!res.ok) {
    let detail = res.statusText;
    try {
      const body = await res.json();
      if (body && typeof body.message === "string") detail = body.message;
      else if (body && typeof body.error === "string") detail = body.error;
    } catch {
      /* ignore body parse failure */
    }
    throw new ApiError(res.status, detail);
  }
  if (res.status === 204 || res.headers.get("content-length") === "0") {
    return undefined as T;
  }
  return (await res.json()) as T;
}

export function get<T>(path: string): Promise<T> {
  return send<T>(path, {});
}

export function post<T>(path: string, body: unknown): Promise<T> {
  return send<T>(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

export function patch<T>(path: string, body: unknown): Promise<T> {
  return send<T>(path, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

export function put<T>(path: string, body: unknown): Promise<T> {
  return send<T>(path, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

export function setCapabilityToken(token: string): void {
  const value = token.trim();
  if (value) sessionStorage.setItem("garden.capability_token", value);
  else sessionStorage.removeItem("garden.capability_token");
}
