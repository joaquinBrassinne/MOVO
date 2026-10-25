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
};

/** El nombre legible de una entrada del historial; una acción que no se conoce se muestra tal cual. */
export function describeAuditAction(action: string): string {
  return ACTION_LABEL[action] ?? action;
}

export function describeAdminFailure(failure: Failure): FormMessage {
  switch (failure.error.code) {
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
      return (
        describeFailure(failure).form ?? {
          title: "No pudimos completar la acción",
          body: "Probá de nuevo en unos minutos.",
        }
      );
  }
}
