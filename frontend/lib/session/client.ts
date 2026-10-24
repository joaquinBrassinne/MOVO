// Llamadas del navegador al BFF (/api/session/*). Sin React: la lógica vive acá y
// el hook solo la conecta con el estado.

import type { ApiError, SessionPayload, SessionResult } from "./types";

type Action = "login" | "register" | "refresh" | "logout";

const NETWORK_ERROR: ApiError = {
  code: "network_error",
  message: "the request could not be completed",
};

async function readError(response: Response): Promise<ApiError> {
  try {
    const body = (await response.json()) as { error?: ApiError };
    if (body.error?.code) return body.error;
  } catch {
    // Cuerpo vacío o que no es JSON: se cae al error genérico de abajo.
  }
  return { code: "unexpected_response", message: "unexpected response from the service" };
}

async function call(
  action: Action,
  credentials?: { email: string; password: string },
): Promise<SessionResult> {
  try {
    const response = await fetch(`/api/session/${action}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: credentials ? JSON.stringify(credentials) : undefined,
      cache: "no-store",
    });

    if (response.ok) {
      return { ok: true, data: (await response.json()) as SessionPayload };
    }

    const retryAfter = Number(response.headers.get("Retry-After"));
    return {
      ok: false,
      status: response.status,
      error: await readError(response),
      ...(retryAfter > 0 ? { retryAfterSeconds: retryAfter } : {}),
    };
  } catch {
    return { ok: false, status: 0, error: NETWORK_ERROR };
  }
}

export const login = (email: string, password: string) => call("login", { email, password });
export const register = (email: string, password: string) => call("register", { email, password });

export async function logout(): Promise<void> {
  await call("logout");
}

let inFlight: Promise<SessionResult> | null = null;

/**
 * Renueva la sesión con la cookie. El refresh token rota en cada uso y reutilizar uno
 * ya rotado cierra todas las sesiones, así que dos renovaciones simultáneas con el mismo
 * token serían un desastre: dentro de la pestaña comparten una sola llamada, y entre
 * pestañas se turnan con un lock para que la segunda ya lea la cookie rotada.
 */
export function refresh(): Promise<SessionResult> {
  inFlight ??= withTabLock(() => call("refresh")).finally(() => {
    inFlight = null;
  });
  return inFlight;
}

function withTabLock<T>(task: () => Promise<T>): Promise<T> {
  if (typeof navigator !== "undefined" && navigator.locks) {
    return navigator.locks.request("movo-session-refresh", task);
  }
  return task();
}
