// Pure rules of the assisted password reset: the new-password checklist, the link the
// admin shares and how each failure reads. Testable without React.

import { PASSWORD_MAX_LENGTH, PASSWORD_MIN_LENGTH } from "../session/messages.ts";

export const RESET_PATH = "/owners/reset-password";
export const RESET_VALID_HOURS = 24;

export interface PasswordChecks {
  longEnough: boolean;
  tooLong: boolean;
  matches: boolean;
  /** Every rule holds, so the form can be sent. */
  valid: boolean;
}

export function checkPasswords(password: string, repeat: string): PasswordChecks {
  const length = [...password].length;
  const longEnough = length >= PASSWORD_MIN_LENGTH;
  const tooLong = length > PASSWORD_MAX_LENGTH;
  const matches = password !== "" && password === repeat;
  return { longEnough, tooLong, matches, valid: longEnough && !tooLong && matches };
}

/** The link the admin sends to the owner; the API returns only the token. */
export function buildResetLink(origin: string, token: string): string {
  return `${origin}${RESET_PATH}?token=${encodeURIComponent(token)}`;
}

/** wa.me link that opens a chat with the owner with the reset link in the message. */
export function resetWhatsAppUrl(whatsapp: string, link: string): string | null {
  const digits = whatsapp.replace(/\D/g, "");
  if (digits === "") return null;
  const text = `Hola, este es tu enlace para elegir una clave nueva en MOVO (vale ${RESET_VALID_HOURS} horas y sirve una sola vez): ${link}`;
  return `https://wa.me/${digits}?text=${encodeURIComponent(text)}`;
}

export interface Message {
  title: string;
  body: string;
}

interface Failure {
  status: number;
  error: { code: string };
}

/** Admin side: why the link could not be generated. */
export function describeIssueFailure(failure: Failure): Message {
  switch (failure.error.code) {
    case "user_not_found":
      return {
        title: "No encontramos al dueño",
        body: "Ningún dueño tiene ese email. Solo se resetea la clave de dueños de locales.",
      };
    case "forbidden":
      return { title: "Sin permiso", body: "Tu cuenta no puede hacer esta acción." };
    default:
      return { title: "No pudimos generar el enlace", body: "Probá de nuevo en unos minutos." };
  }
}

export type ResetOutcome = "invalid_token" | "too_many" | "password" | "generic";

export function classifyResetFailure(failure: Failure): ResetOutcome {
  switch (failure.error.code) {
    case "invalid_reset_token":
      return "invalid_token";
    case "too_many_attempts":
      return "too_many";
    case "validation_failed":
      return "password";
    default:
      return "generic";
  }
}

/** Owner side: a message for the failures that keep the form on screen. */
export function describeResetFailure(outcome: ResetOutcome): Message {
  switch (outcome) {
    case "too_many":
      return {
        title: "Demasiados intentos",
        body: "Esperá unos minutos antes de probar de nuevo.",
      };
    case "password":
      return {
        title: "Revisá la clave",
        body: `Usá entre ${PASSWORD_MIN_LENGTH} y ${PASSWORD_MAX_LENGTH} caracteres.`,
      };
    default:
      return { title: "No pudimos guardar tu clave", body: "Probá de nuevo en unos minutos." };
  }
}
