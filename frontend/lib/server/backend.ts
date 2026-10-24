// Cliente del backend Go para código que corre en el servidor de Next.
// El navegador nunca llama a la API de autenticación directamente: pasa por
// los route handlers de app/api/session, que son los únicos que ven el refresh token.

const API_URL = process.env.API_URL ?? "http://localhost:8080";
const TIMEOUT_MS = 10_000;

export type BackendResponse = {
  status: number;
  body: unknown;
  retryAfter: string | null;
};

const UPSTREAM_UNAVAILABLE = {
  error: {
    code: "upstream_unavailable",
    message: "the service is not reachable right now",
  },
};

/** POST JSON a `/api/v1<path>`. Una caída de red se informa como 502 con el mismo sobre de error. */
export async function postToBackend(path: string, payload: unknown): Promise<BackendResponse> {
  try {
    const response = await fetch(`${API_URL}/api/v1${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
      cache: "no-store",
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });

    const text = await response.text();
    return {
      status: response.status,
      body: text === "" ? null : (JSON.parse(text) as unknown),
      retryAfter: response.headers.get("Retry-After"),
    };
  } catch {
    return { status: 502, body: UPSTREAM_UNAVAILABLE, retryAfter: null };
  }
}
