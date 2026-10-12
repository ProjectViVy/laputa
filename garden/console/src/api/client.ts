let activeProfile = "";
export function setActiveProfile(id: string) { activeProfile = id; }

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string, public code?: string, public details?: Record<string, unknown>) {
    super(message);
    this.status = status;
  }
}

export async function sendForProfile<T>(profile: string, path: string, init: RequestInit): Promise<T> {
 const res = await fetch(path, {
    ...init,
    credentials: "same-origin",
    headers: { Accept: "application/json", ...(profile ? { "X-Garden-Profile": profile } : {}), ...(init.headers ?? {}) },
  });

 if (!res.ok) {
    let detail = res.statusText;
    let code: string | undefined;
    let details: Record<string, unknown> | undefined;
    try {
      const body = await res.json();
      code = body?.code;
      details = body?.details;
      if (body && typeof body.message === "string") detail = body.message;
      else if (body && typeof body.error === "string") detail = body.error;
    } catch {
      /* ignore body parse failure */
    }
    throw new ApiError(res.status, detail, code, details);
  }
  if (res.status === 204 || res.headers.get("content-length") === "0") {
    return undefined as T;
  }
  return (await res.json()) as T;
}

async function send<T>(path: string, init: RequestInit): Promise<T> {
  const profile = activeProfile;
  const result = await sendForProfile<T>(profile, path, init);
  if (profile !== activeProfile) throw new ApiError(409, "Profile changed; response discarded");
  return result;
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
