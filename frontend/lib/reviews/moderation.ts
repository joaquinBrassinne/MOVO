// Pure rules of the review moderation view: which decisions a status allows, how a
// failure is worded, and the small formatting helpers. Tested without React.

import type { FormMessage } from "../business/messages.ts";
import type { AdminDecision, ModerationReview } from "./types.ts";

export const COMMENT_PREVIEW = 80;

/** "Dejar visible" applies to a reported or hidden review, "Ocultar" to a reported one. */
export function allowedDecisions(
  status: ModerationReview["status"],
): Record<AdminDecision, boolean> {
  return {
    visible: status !== "visible",
    hidden: status !== "hidden",
  };
}

/** The same list without the review that just left the queue. */
export function withoutReview<T extends { id: number }>(reviews: T[], id: number): T[] {
  return reviews.filter((review) => review.id !== id);
}

/** The comment cut to `max` characters with an ellipsis; "" when there is none. */
export function truncateComment(comment: string | undefined, max = COMMENT_PREVIEW): string {
  if (!comment) return "";
  const text = comment.trim();
  return text.length <= max ? text : `${text.slice(0, max).trimEnd()}…`;
}

const MONTHS = ["ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"];

/** "29 oct 2026", in local time. Empty for an invalid date. */
export function formatReviewDate(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return "";
  return `${date.getDate()} ${MONTHS[date.getMonth()]} ${date.getFullYear()}`;
}

export const STATUS_TEXT: Record<ModerationReview["status"], string> = {
  visible: "Visible",
  reported: "Reportada",
  hidden: "Oculta",
};

/** What the admin reads when a decision fails. Never the server's own text. */
export function describeModerationFailure(error: { code: string }): FormMessage {
  switch (error.code) {
    case "review_purged":
      return {
        title: "Esta reseña ya no se puede restaurar",
        body: "Pasaron más de 72 horas desde que se ocultó: su texto y su nombre se borraron.",
      };
    case "review_not_found":
      return { title: "La reseña no existe", body: "Se actualizó la lista." };
    case "forbidden":
      return { title: "Sin permiso", body: "Tu cuenta no puede moderar reseñas." };
    default:
      return {
        title: "No pudimos completar la acción",
        body: "Probá de nuevo en unos minutos.",
      };
  }
}
