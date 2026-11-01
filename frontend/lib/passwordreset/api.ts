import { authed } from "@/lib/api/authed";

import type { IssuedReset } from "./types";

export const issuePasswordReset = (email: string) =>
  authed<IssuedReset>("POST", "/admin/password-resets", { email });

/**
 * POST /auth/reset-password answers 204 with no body, which the shared anonymous helper
 * would read as a network failure, so this call reads the status itself.
 */
export async function resetPassword(token: string, password: string) {
  try {
    const response = await fetch("/api/v1/auth/reset-password", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token, password }),
      cache: "no-store",
      // The token must not leak through the Referer header.
      referrerPolicy: "no-referrer",
    });
    if (response.ok) return { ok: true as const };
    const body = (await response.json().catch(() => ({}))) as { error?: { code?: string } };
    return {
      ok: false as const,
      status: response.status,
      error: { code: body.error?.code ?? "unexpected_response" },
    };
  } catch {
    return { ok: false as const, status: 0, error: { code: "network_error" } };
  }
}
