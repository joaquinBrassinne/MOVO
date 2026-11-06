// Qué se le dice al cliente cuando no puede generar un cupón. Función pura: se prueba sin
// React. Nunca se muestra el texto del error del servidor, solo estos mensajes.

import { formatResetDate } from "../benefits/quota.ts";
import { nextQuotaReset } from "./format.ts";

export const GENERATE_PROBLEM = {
  PENDING: "pending",
  QUOTA: "quota",
  UNAVAILABLE: "unavailable",
  OTHER: "other",
} as const;

export type GenerateProblemKind = (typeof GENERATE_PROBLEM)[keyof typeof GENERATE_PROBLEM];

export interface GenerateFeedback {
  /** Mensajes junto al campo al que se refieren: `name` o `phone`. */
  fields: Record<string, string>;
  /** Un estado de la generación que merece su propio aviso. */
  problem?: { kind: GenerateProblemKind; title: string; body: string };
}

const FIELD_MESSAGES: Record<string, string> = {
  name: "Ingresá tu nombre",
  phone: "Ingresá tu número con característica, sin 0 ni 15",
};

export function describeGenerateFailure(
  error: { code: string; details?: unknown },
  now: Date,
): GenerateFeedback {
  switch (error.code) {
    case "validation_failed": {
      const { field } = (error.details ?? {}) as { field?: string };
      const message = field ? FIELD_MESSAGES[field] : undefined;
      return field && message ? { fields: { [field]: message } } : other();
    }

    case "pending_coupon":
      return problem(
        GENERATE_PROBLEM.PENDING,
        "Ya tenés un cupón activo en este local",
        "Podés mostrarlo directamente desde tu dispositivo al llegar.",
      );

    case "quota_reached": {
      // El cupo vuelve a empezar el primer día del mes siguiente (hora de Argentina).
      const reset = formatResetDate(nextQuotaReset(now));
      return problem(
        GENERATE_PROBLEM.QUOTA,
        "Cupos mensuales alcanzados",
        `Este local llegó a su límite de cupones de este mes. Nuevos lugares disponibles el ${reset}.`,
      );
    }

    case "benefit_not_found":
      return problem(
        GENERATE_PROBLEM.UNAVAILABLE,
        "Este beneficio ya no está disponible",
        "Volvé a la ficha del local para ver los que tiene ahora.",
      );

    case "rate_limited": {
      const { retry_after_seconds: seconds } = (error.details ?? {}) as {
        retry_after_seconds?: number;
      };
      const minutes = seconds ? Math.max(1, Math.ceil(seconds / 60)) : null;
      return problem(
        GENERATE_PROBLEM.OTHER,
        "Hiciste muchos intentos",
        minutes
          ? `Probá de nuevo en ${minutes} ${minutes === 1 ? "minuto" : "minutos"}.`
          : "Probá de nuevo en un rato.",
      );
    }

    case "network_error":
      return problem(
        GENERATE_PROBLEM.OTHER,
        "Sin conexión",
        "Revisá tu conexión a internet y probá de nuevo.",
      );

    default:
      return other();
  }
}

function problem(kind: GenerateProblemKind, title: string, body: string): GenerateFeedback {
  return { fields: {}, problem: { kind, title, body } };
}

function other(): GenerateFeedback {
  return problem(
    GENERATE_PROBLEM.OTHER,
    "No pudimos generar el cupón",
    "Probá de nuevo en unos minutos.",
  );
}
