// Pure rules of the owner's reviews list: what a report changes on screen and how a failed
// report is worded. Tested without React.

import type { FormMessage } from "../business/messages.ts";
import type { ModerationReview } from "./types.ts";

/** The same list with that review marked as reported; the others are untouched. */
export function markReported(reviews: ModerationReview[], id: number): ModerationReview[] {
  return reviews.map((review) => (review.id === id ? { ...review, status: "reported" } : review));
}

/** Only a visible review can be reported: a reported one is already with the team. */
export function canReport(review: Pick<ModerationReview, "status">): boolean {
  return review.status === "visible";
}

/** What the owner reads when a report fails. Never the server's own text. */
export function describeReportFailure(error: { code: string }): FormMessage {
  switch (error.code) {
    case "review_not_found":
      return { title: "La reseña no existe", body: "Se actualizó la lista." };
    case "business_deleted":
      return {
        title: "Tu negocio fue eliminado",
        body: "Ya no podés reportar reseñas desde esta cuenta.",
      };
    default:
      return {
        title: "No pudimos reportar la reseña",
        body: "Probá de nuevo en unos minutos.",
      };
  }
}
