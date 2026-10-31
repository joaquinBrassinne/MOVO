// Cómo se muestra cada error de las acciones del super admin y cómo se nombra cada
// entrada del historial. Funciones puras: se prueban sin React.

import { describeFailure, type Failure, type FormMessage } from "../business/messages.ts";
import { BUSINESS_STATUS, type BusinessStatus } from "../business/types.ts";

export const STATUS_LABEL: Record<BusinessStatus, string> = {
  [BUSINESS_STATUS.DRAFT]: "Borrador",
  [BUSINESS_STATUS.PENDING_REVIEW]: "En revisión",
  [BUSINESS_STATUS.ACTIVE]: "Activo",
  [BUSINESS_STATUS.PAUSED]: "Pausado",
  [BUSINESS_STATUS.DELETED]: "Baja",
};

const ACTION_LABEL: Record<string, string> = {
  "business.approve": "Aprobado y publicado",
  "business.pause": "Pausado",
  "business.reactivate": "Reactivado",
  "business.delete": "Dado de baja",
  "business.verify": "Marcado como verificado",
  "business.revoke_verification": "Verificación quitada",
  "business.mark_pilot": "Marcado como piloto",
  "business.unmark_pilot": "Piloto quitado",
  "business.profile_update": "Datos del negocio actualizados",
  "billing.confirm_payment": "Pago confirmado y local publicado",
  "billing.register_payment": "Pago mensual registrado",
  "billing.change_plan": "Plan cambiado",
};

const BILLING_FIELD: Record<string, string> = {
  plan: "Elegí un plan válido (Starter o Pro).",
  mp_subscription_ref: "La referencia de Mercado Pago es demasiado larga o no es válida.",
  notes: "Las notas son demasiado largas.",
  status: "El estado de la suscripción no es válido.",
};

const PAID_UNTIL_REASON: Record<string, string> = {
  invalid: "La vigencia inicial no es una fecha válida.",
  in_the_past: "La vigencia inicial no puede ser anterior a hoy.",
};

function describeValidation(details: unknown): FormMessage | null {
  const { field, reason } = (details ?? {}) as { field?: string; reason?: string };
  const body =
    field === "paid_until"
      ? PAID_UNTIL_REASON[reason ?? ""]
      : field === "notes" && reason === "too_long"
        ? "Las notas son demasiado largas (máximo 500 caracteres)."
        : field
          ? BILLING_FIELD[field]
          : undefined;
  return body ? { title: "Revisá los datos", body } : null;
}

/** El nombre legible de una entrada del historial; una acción que no se conoce se muestra tal cual. */
export function describeAuditAction(action: string): string {
  return ACTION_LABEL[action] ?? action;
}

export function describeAdminFailure(failure: Failure): FormMessage {
  switch (failure.error.code) {
    case "business_not_found":
      return { title: "No encontramos el local", body: "Puede haber sido dado de baja." };
    case "subscription_not_found":
      return {
        title: "Este local no tiene suscripción",
        body: "Confirmá su pago para crearla.",
      };
    case "business_not_confirmable":
      return {
        title: "No se puede confirmar el pago de este local",
        body: "Solo se confirma el pago de un local en revisión o activo. Se recargaron sus datos.",
      };
    case "subscription_exists":
      return {
        title: "Este local ya tiene suscripción",
        body: "Para sumar un mes usá “Registrar pago”. Se recargaron sus datos.",
      };
    case "validation_failed": {
      const message = describeValidation(failure.error.details);
      if (message) return message;
      break;
    }
    case "invalid_state":
      return {
        title: "Ese cambio no corresponde",
        body: "El estado actual del local no permite esta acción. Se recargaron sus datos.",
      };
    case "conflict":
      return {
        title: "Alguien cambió este local antes",
        body: "Se recargaron sus datos. Revisalos y volvé a intentar.",
      };
    case "no_change":
      return { title: "Ya estaba así", body: "No hubo nada que cambiar." };
    case "business_deleted":
      return { title: "El local fue dado de baja", body: "Ya no se puede modificar." };
    case "forbidden":
      return { title: "Sin permiso", body: "Tu cuenta no puede hacer esta acción." };
    default:
      break;
  }
  return (
    describeFailure(failure).form ?? {
      title: "No pudimos completar la acción",
      body: "Probá de nuevo en unos minutos.",
    }
  );
}
