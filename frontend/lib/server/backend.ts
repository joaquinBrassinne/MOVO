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

// Rutas de la API que el navegador puede usar a través del proxy de app/api/v1. La lista
// es cerrada a propósito: /auth/* no está, porque ahí viaja el refresh token y solo debe
// verlo app/api/session (cookie httpOnly).
const PROXIED = [
  /^me\//,
  /^admin\//,
  /^plans$/,
  /^categories$/,
  // Búsqueda pública (Home y Mapa): sin identidad, solo lectura.
  /^search$/,
  /^home\/featured$/,
  // La ficha del local: el aviso de vista (POST) y el botón de WhatsApp, que cuenta el clic
  // y redirige a wa.me. La respuesta del botón es una redirección que el navegador sigue.
  /^businesses\/[a-z0-9-]+\/view$/,
  /^r\/wa\/\d+$/,
  // El cliente no tiene cuenta: genera su cupón (POST) y lo vuelve a ver con su enlace de
  // acceso, cuyo token tiene 43 caracteres. Son los únicos puntos de cupones sin sesión.
  /^benefits\/\d+\/codes$/,
  /^access\/[A-Za-z0-9_-]{43}$/,
];

/**
 * Cabeceras que el navegador puede mandar a la API; el resto no se reenvía. El user-agent
 * viaja para que el backend descarte bots en las vistas y los clics: sin él vería al proxy.
 */
const FORWARDED_REQUEST_HEADERS = ["authorization", "content-type", "accept", "user-agent"];
const FORWARDED_RESPONSE_HEADERS = [
  "content-type",
  "cache-control",
  "retry-after",
  "location",
  "referrer-policy",
];

const METHODS_WITHOUT_BODY = new Set(["GET", "HEAD", "DELETE"]);

function notFound(): Response {
  return Response.json(
    { error: { code: "not_found", message: "resource not found" } },
    { status: 404 },
  );
}

/**
 * Reenvía una petición del navegador a `/api/v1/<path>` del backend con su token. Solo
 * pasa lo permitido por PROXIED, sin segmentos raros, y la respuesta vuelve tal cual.
 */
export async function proxyToBackend(request: Request, segments: string[]): Promise<Response> {
  const unsafe = segments.some((s) => s === "" || s === "." || s === ".." || /[\/%]/.test(s));
  const path = segments.join("/");
  if (unsafe || !PROXIED.some((allowed) => allowed.test(path))) return notFound();

  const headers = new Headers();
  for (const name of FORWARDED_REQUEST_HEADERS) {
    const value = request.headers.get(name);
    if (value) headers.set(name, value);
  }

  try {
    const upstream = await fetch(`${API_URL}/api/v1/${path}${new URL(request.url).search}`, {
      method: request.method,
      headers,
      body: METHODS_WITHOUT_BODY.has(request.method) ? undefined : await request.arrayBuffer(),
      cache: "no-store",
      // Una redirección (el botón de WhatsApp) la sigue el navegador, no el proxy.
      redirect: "manual",
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });

    const responseHeaders = new Headers();
    for (const name of FORWARDED_RESPONSE_HEADERS) {
      const value = upstream.headers.get(name);
      if (value) responseHeaders.set(name, value);
    }
    return new Response(upstream.status === 204 ? null : upstream.body, {
      status: upstream.status,
      headers: responseHeaders,
    });
  } catch {
    return Response.json(UPSTREAM_UNAVAILABLE, { status: 502 });
  }
}
