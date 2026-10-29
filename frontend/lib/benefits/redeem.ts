// Qué se le dice al dueño cuando un cupón no se puede canjear. Función pura: se prueba sin
// React. Nunca se muestra el texto del error del servidor, solo estos mensajes.

import { COUPON_STATUS, type CouponStatus } from "./types.ts";

export const PROBLEM = {
  NOT_FOUND: "not_found",
  EXPIRED: "expired",
  REDEEMED: "redeemed",
  BRANCH: "branch",
  OTHER: "other",
} as const;

export type ProblemKind = (typeof PROBLEM)[keyof typeof PROBLEM];

export interface Problem {
  kind: ProblemKind;
  title: string;
  body: string;
}

const NOT_FOUND: Problem = {
  kind: PROBLEM.NOT_FOUND,
  title: "No encontramos ese cupón",
  // La API no distingue un código que no existe de uno de otro negocio: no se puede
  // sondear qué códigos tienen los demás, así que los dos se explican igual.
  body: "No existe o no es de tu negocio. Revisá el código con tu cliente.",
};

const EXPIRED: Problem = {
  kind: PROBLEM.EXPIRED,
  title: "Cupón vencido",
  body: "Pasaron las 48 horas desde que se generó. Tu cliente puede generar otro.",
};

const REDEEMED: Problem = {
  kind: PROBLEM.REDEEMED,
  title: "Ya fue canjeado",
  body: "Este cupón ya se usó y no se puede canjear de nuevo.",
};

const BRANCH: Problem = {
  kind: PROBLEM.BRANCH,
  title: "Elegí la sede",
  body: "Indicá en qué sede se canjea este cupón.",
};

const OTHER: Problem = {
  kind: PROBLEM.OTHER,
  title: "No pudimos completar el canje",
  body: "Probá de nuevo en unos minutos.",
};

/** El problema de una respuesta de error de la API al buscar o canjear un cupón. */
export function problemOf(error: { code: string; details?: unknown }): Problem {
  switch (error.code) {
    case "coupon_not_found":
      return NOT_FOUND;
    case "coupon_expired":
      return EXPIRED;
    case "coupon_redeemed":
      return REDEEMED;
    case "validation_failed": {
      const { field } = (error.details ?? {}) as { field?: string };
      return field === "branch_id" ? BRANCH : OTHER;
    }
    case "network_error":
      return {
        kind: PROBLEM.OTHER,
        title: "Sin conexión",
        body: "Revisá tu conexión a internet y probá de nuevo.",
      };
    case "unauthorized":
    case "invalid_session":
      return {
        kind: PROBLEM.OTHER,
        title: "Tu sesión venció",
        body: "Ingresá de nuevo para continuar.",
      };
    default:
      return OTHER;
  }
}

/** Un cupón que ya se vio pero no se puede canjear: vencido o ya canjeado; `null` si se puede. */
export function problemOfStatus(status: CouponStatus): Problem | null {
  if (status === COUPON_STATUS.EXPIRED) return EXPIRED;
  if (status === COUPON_STATUS.REDEEMED) return REDEEMED;
  return null;
}

/** "ya canjeó 3 cupones en tu negocio" / "es la primera vez que canjea en tu negocio". */
/**
 * Cuántos canjes anteriores quedan fuera de la lista del cupón. El servidor cuenta todos los
 * canjes del cliente en el negocio, incluido este cupón si ya está canjeado, pero la lista
 * trae solo los más nuevos y nunca al propio cupón.
 */
export function olderRedemptions(preview: {
  status: CouponStatus;
  customer_redeemed_count: number;
  previous_redemptions: readonly unknown[];
}): number {
  const others =
    preview.customer_redeemed_count - (preview.status === COUPON_STATUS.REDEEMED ? 1 : 0);
  return Math.max(0, others - preview.previous_redemptions.length);
}

export function describeReturning(redeemedBefore: number): string {
  if (redeemedBefore <= 0) return "Es la primera vez que este cliente canjea en tu negocio.";
  return `Este cliente ya canjeó ${redeemedBefore} ${
    redeemedBefore === 1 ? "cupón" : "cupones"
  } en tu negocio.`;
}
