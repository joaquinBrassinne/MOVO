// Contrato de /api/session/* (el BFF de Next) y del sobre de error de la API Go.

export type SessionUser = {
  id: number;
  email: string;
  role: "business_admin" | "super_admin";
};

/** Lo que el navegador recibe: el refresh token nunca viaja hasta acá. */
export type SessionPayload = {
  user: SessionUser;
  access_token: string;
  access_expires_at: string;
};

export type ApiError = {
  code: string;
  message: string;
  details?: unknown;
};

export type SessionResult =
  | { ok: true; data: SessionPayload }
  | { ok: false; status: number; error: ApiError; retryAfterSeconds?: number };
