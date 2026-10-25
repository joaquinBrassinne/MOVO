// Peticiones a la API con la sesión del portal: ponen el access token, lo renuevan si
// hace falta y reintentan una vez si la API lo rechaza.

import { getAccessToken, refresh } from "@/lib/session/client";

import { request, type ApiResult } from "./request";

const NO_SESSION = {
  ok: false,
  status: 401,
  error: { code: "unauthorized", message: "authentication is required" },
} as const;

export async function authed<T>(
  method: string,
  path: string,
  body?: unknown,
): Promise<ApiResult<T>> {
  const token = await getAccessToken();
  if (!token) return NO_SESSION;

  const first = await request<T>(method, path, token, body);
  if (first.ok || first.status !== 401) return first;

  // El token pudo vencer o revocarse entre que se eligió y se usó: se renueva una vez.
  const renewed = await refresh();
  if (!renewed.ok) return first;
  return request<T>(method, path, renewed.data.access_token, body);
}
