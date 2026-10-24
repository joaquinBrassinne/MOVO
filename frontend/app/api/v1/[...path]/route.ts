import { proxyToBackend } from "@/lib/server/backend";

// Proxy del navegador a la API Go (ver lib/server/backend.ts para qué rutas pasan).
// Lleva el access token que manda el cliente; nunca ve ni guarda el refresh token.
async function handle(request: Request, ctx: RouteContext<"/api/v1/[...path]">) {
  const { path } = await ctx.params;
  return proxyToBackend(request, path);
}

export const GET = handle;
export const POST = handle;
export const PUT = handle;
export const DELETE = handle;
