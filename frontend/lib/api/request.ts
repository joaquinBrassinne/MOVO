// Llamadas del navegador a la API Go (por las rewrites de next.config.ts). Sin React.

import type { ApiError } from "@/lib/session/types";

export type ApiResult<T> = { ok: true; data: T } | { ok: false; status: number; error: ApiError };

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

/** Una petición JSON a `/api/v1<path>` con el access token. Un 204 devuelve `undefined`. */
export async function request<T>(
  method: string,
  path: string,
  token: string,
  body?: unknown,
): Promise<ApiResult<T>> {
  try {
    const response = await fetch(`/api/v1${path}`, {
      method,
      headers: {
        Authorization: `Bearer ${token}`,
        ...(body === undefined ? {} : { "Content-Type": "application/json" }),
      },
      body: body === undefined ? undefined : JSON.stringify(body),
      cache: "no-store",
    });

    if (!response.ok)
      return { ok: false, status: response.status, error: await readError(response) };
    if (response.status === 204) return { ok: true, data: undefined as T };
    return { ok: true, data: (await response.json()) as T };
  } catch {
    return { ok: false, status: 0, error: NETWORK_ERROR };
  }
}

/** Lo mismo para los endpoints públicos (/plans, /categories): no llevan token. */
export async function requestPublic<T>(path: string): Promise<ApiResult<T>> {
  try {
    const response = await fetch(`/api/v1${path}`, { cache: "no-store" });
    if (!response.ok)
      return { ok: false, status: response.status, error: await readError(response) };
    return { ok: true, data: (await response.json()) as T };
  } catch {
    return { ok: false, status: 0, error: NETWORK_ERROR };
  }
}

/**
 * Una petición JSON sin sesión, para lo que hace un cliente que no tiene cuenta (generar un
 * cupón, ver el que guardó). El token del cupón, si lo hay, va en la ruta.
 */
export async function requestAnonymous<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<ApiResult<T>> {
  try {
    const response = await fetch(`/api/v1${path}`, {
      method,
      headers: body === undefined ? {} : { "Content-Type": "application/json" },
      body: body === undefined ? undefined : JSON.stringify(body),
      cache: "no-store",
      // Una ruta con un token no debe filtrarse a otro sitio en la cabecera Referer.
      referrerPolicy: "no-referrer",
    });

    if (!response.ok)
      return { ok: false, status: response.status, error: await readError(response) };
    return { ok: true, data: (await response.json()) as T };
  } catch {
    return { ok: false, status: 0, error: NETWORK_ERROR };
  }
}
