// Pure rules of the review form: labels, limits and how a failed submit is told to the
// customer. No React and no aliased imports, so they are tested without a bundler.

import { formatShortDate } from "../coupons/format.ts";
import type { ReviewFormState } from "./types.ts";

export const COMMENT_MAX = 500;
export const ANONYMOUS_NAME = "Cliente de MOVO";

const RATING_LABELS = ["", "Mala", "Regular", "Buena", "¡Muy buena!", "¡Excelente!"];

/** "4 de 5 · ¡Muy buena!" for a picked rating; empty for none. */
export function ratingLabel(rating: number): string {
  const label = RATING_LABELS[rating];
  return label ? `${rating} de 5 · ${label}` : "";
}

/** "24 oct": the short day used on the form. */
export function formatDay(iso: string | null): string {
  return iso ? formatShortDate(iso).replace(" de ", " ") : "";
}

/** The comment cut to the limit; the counter and the field never go past it. */
export function clampComment(text: string): string {
  return Array.from(text).slice(0, COMMENT_MAX).join("");
}

export const commentLength = (text: string): number => Array.from(text).length;

/** "Branch, City", whichever parts exist. */
export function placeLine(branch: string | null, city: string | null): string {
  return [branch, city].filter(Boolean).join(", ");
}

export type SubmitFailure =
  | { kind: "state"; state: ReviewFormState }
  | { kind: "rating" | "comment" | "generic"; message: string };

const GENERIC = "No pudimos publicar tu reseña. Probá de nuevo en unos minutos.";

/** Maps the API error of a submit to what the screen does. Never shows the server text. */
export function describeSubmitFailure(error: { code: string; details?: unknown }): SubmitFailure {
  switch (error.code) {
    case "already_reviewed":
      return { kind: "state", state: "reviewed" };
    case "review_window_closed":
      return { kind: "state", state: "window_closed" };
    case "coupon_not_redeemed":
      return { kind: "state", state: "not_redeemed" };
    case "validation_failed": {
      const field = (error.details as { field?: string } | undefined)?.field;
      if (field === "comment")
        return {
          kind: "comment",
          message: "Tu comentario no puede incluir teléfonos, enlaces ni datos de contacto.",
        };
      if (field === "rating") return { kind: "rating", message: "Elegí de 1 a 5 estrellas." };
      return { kind: "generic", message: GENERIC };
    }
    default:
      return { kind: "generic", message: GENERIC };
  }
}
