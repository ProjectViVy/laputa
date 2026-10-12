let activeProfile = "";
export function setActiveProfile(id: string) { activeProfile = id; }

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function send<T>(path: string, init: RequestInit): Promise<T> {
  const profile = activeProfile;
 const res = await fetch(path, {
    ...init,
    credentials: "same-origin",
    headers: { Accept: "application/json", ...(profile ? { "X-Garden-Profile": profile } : {}), ...(init.headers ?? {}) },
  });
  if (profile !== activeProfile) throw new ApiError(409, "Profile changed; response discarded");
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

export function get<T>(path: string, signal?: AbortSignal): Promise<T> {
  return send<T>(path, { signal });
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

export function remove<T>(path: string): Promise<T> { return send<T>(path, { method: "DELETE" }); }
