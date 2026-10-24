// Cómo se muestra cada error de /api/session en las vistas de acceso. Es una función
// pura para poder probarla sin React.

import type { SessionResult } from "./types";

/** Espejo de domain.ValidatePassword en el backend (longitud en caracteres). */
export const PASSWORD_MIN_LENGTH = 10;
export const PASSWORD_MAX_LENGTH = 128;

export type AuthFeedback = {
  /** Mensaje del formulario completo (credenciales, límites, caídas). */
  form?: { title: string; body: string };
  /** Mensajes junto al campo que corresponde. */
  fields: { email?: string; password?: string };
};

type Failure = Extract<SessionResult, { ok: false }>;

export function describeAuthFailure(failure: Failure): AuthFeedback {
  const { error, retryAfterSeconds } = failure;

  switch (error.code) {
    case "invalid_credentials":
      return {
        fields: {},
        form: {
          title: "Email o contraseña incorrectos",
          body: "Verificá tus datos o, si olvidaste tu clave, pedí ayuda desde el link inferior.",
        },
      };

    case "too_many_attempts": {
      const minutes = Math.max(1, Math.ceil((retryAfterSeconds ?? 60) / 60));
      return {
        fields: {},
        form: {
          title: "Demasiados intentos",
          body: `Esperá ${minutes} ${minutes === 1 ? "minuto" : "minutos"} y volvé a probar.`,
        },
      };
    }

    case "email_taken":
      return { fields: { email: "Ese email ya está registrado" } };

    case "validation_failed":
      return { fields: validationFields(error.details) };

    default:
      return {
        fields: {},
        form: {
          title: "No pudimos completar la acción",
          body: "Probá de nuevo en unos minutos.",
        },
      };
  }
}

function validationFields(details: unknown): AuthFeedback["fields"] {
  const { field, reason } = (details ?? {}) as { field?: string; reason?: string };

  if (field === "email") return { email: "Ingresá un email válido" };
  if (field === "password" && reason === "too_long") {
    return { password: `Máximo ${PASSWORD_MAX_LENGTH} caracteres` };
  }
  return { password: `Mínimo ${PASSWORD_MIN_LENGTH} caracteres` };
}
