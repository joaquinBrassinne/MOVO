import { cookies } from "next/headers";

import { postToBackend, type BackendResponse } from "@/lib/server/backend";

// El refresh token vive solo en esta cookie httpOnly, limitada a /api/session:
// el JavaScript de la página no puede leerlo y el resto del sitio no lo recibe.
const REFRESH_COOKIE = "movo_refresh";
const COOKIE_PATH = "/api/session";

const BACKEND_PATH = {
  login: "/auth/login",
  register: "/auth/register",
  refresh: "/auth/refresh",
  logout: "/auth/logout",
} as const;

type Action = keyof typeof BACKEND_PATH;

function isAction(value: string): value is Action {
  return Object.hasOwn(BACKEND_PATH, value);
}

function failure(status: number, code: string, message: string): Response {
  return Response.json({ error: { code, message } }, { status });
}

/** Solo estos dos campos se reenvían: nada que el cliente agregue llega a la API. */
async function readCredentials(
  request: Request,
): Promise<{ email: string; password: string } | null> {
  if (!request.headers.get("content-type")?.startsWith("application/json")) return null;

  try {
    const body: unknown = await request.json();
    if (typeof body !== "object" || body === null) return null;

    const { email, password } = body as Record<string, unknown>;
    if (typeof email !== "string" || typeof password !== "string") return null;

    return { email, password };
  } catch {
    return null;
  }
}

type SessionBody = {
  user: unknown;
  access_token: string;
  access_expires_at: string;
  refresh_token: string;
  refresh_expires_at: string;
};

function isSessionBody(body: unknown): body is SessionBody {
  if (typeof body !== "object" || body === null) return false;
  const candidate = body as Record<string, unknown>;
  return (
    typeof candidate.access_token === "string" &&
    typeof candidate.access_expires_at === "string" &&
    typeof candidate.refresh_token === "string" &&
    typeof candidate.refresh_expires_at === "string"
  );
}

/** Guarda el refresh token en la cookie y devuelve al navegador el resto de la sesión. */
async function respondWithSession(upstream: BackendResponse): Promise<Response> {
  if (!isSessionBody(upstream.body)) {
    return failure(502, "upstream_unavailable", "unexpected response from the service");
  }

  const { user, access_token, access_expires_at, refresh_token, refresh_expires_at } =
    upstream.body;

  (await cookies()).set(REFRESH_COOKIE, refresh_token, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: COOKIE_PATH,
    expires: new Date(refresh_expires_at),
  });

  return Response.json({ user, access_token, access_expires_at }, { status: upstream.status });
}

function passThrough(upstream: BackendResponse): Response {
  const headers = new Headers();
  if (upstream.retryAfter) headers.set("Retry-After", upstream.retryAfter);

  return Response.json(upstream.body, { status: upstream.status, headers });
}

export async function POST(request: Request, ctx: RouteContext<"/api/session/[action]">) {
  const { action } = await ctx.params;
  if (!isAction(action)) return failure(404, "not_found", "resource not found");

  const store = await cookies();
  const refreshToken = store.get(REFRESH_COOKIE)?.value;

  if (action === "login" || action === "register") {
    const credentials = await readCredentials(request);
    if (!credentials)
      return failure(400, "invalid_request", "request body is not valid for this endpoint");

    const upstream = await postToBackend(BACKEND_PATH[action], credentials);
    return upstream.status < 300 ? respondWithSession(upstream) : passThrough(upstream);
  }

  if (action === "refresh") {
    if (!refreshToken) return failure(401, "invalid_session", "there is no session to renew");

    const upstream = await postToBackend(BACKEND_PATH.refresh, { refresh_token: refreshToken });
    if (upstream.status < 300) return respondWithSession(upstream);

    // Rechazado de forma definitiva: la cookie ya no sirve. Una caída o un límite no la tocan.
    if (upstream.status === 401) store.delete({ name: REFRESH_COOKIE, path: COOKIE_PATH });
    return passThrough(upstream);
  }

  // logout: es idempotente y la cookie se borra pase lo que pase con la API.
  if (refreshToken) await postToBackend(BACKEND_PATH.logout, { refresh_token: refreshToken });
  store.delete({ name: REFRESH_COOKIE, path: COOKIE_PATH });
  return new Response(null, { status: 204 });
}
