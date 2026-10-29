"use client";

import { useState } from "react";

import { Notice } from "@/components/onboarding/fields";
import { StarRating } from "@/components/reviews/StarRating";
import { XIcon } from "@/components/socios/icons";
import type { FormMessage } from "@/lib/business/messages";
import {
  MIN_REVIEWS_TO_RATE,
  type AdminDecision,
  type ModerationReview,
} from "@/lib/reviews/types";

import { allowedDecisions, formatReviewDate, STATUS_TEXT } from "@/lib/reviews/moderation";

interface Props {
  review: ModerationReview;
  onClose: () => void;
  onDecide: (id: number, decision: AdminDecision) => Promise<FormMessage | null>;
}

/** Detalle de una reseña: panel lateral en escritorio, hoja desde abajo en el celular. */
export function ReviewDetail({ review, onClose, onDecide }: Props) {
  const [pending, setPending] = useState<AdminDecision | null>(null);
  const [message, setMessage] = useState<FormMessage | null>(null);
  const allowed = allowedDecisions(review.status);

  async function decide(decision: AdminDecision) {
    if (pending) return;
    setPending(decision);
    setMessage(null);
    const failure = await onDecide(review.id, decision);
    // On success the row leaves the list and this panel unmounts with it.
    setMessage(failure);
    setPending(null);
  }

  return (
    <>
      <div
        aria-hidden="true"
        onClick={onClose}
        className="fixed inset-0 z-30 bg-black/40 xl:hidden"
      />
      <aside
        aria-label={`Detalle de la reseña de ${review.business_name}`}
        className="fixed inset-x-0 bottom-0 z-40 max-h-[85dvh] space-y-5 overflow-y-auto rounded-t-3xl bg-white p-6 shadow-lg xl:sticky xl:inset-auto xl:top-6 xl:z-auto xl:max-h-none xl:rounded-3xl xl:shadow-sm"
      >
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <span className="inline-flex rounded-full bg-movo-accent/40 px-3 py-1 text-xs font-bold">
              {STATUS_TEXT[review.status]}
            </span>
            <h2 className="mt-2 font-display text-2xl font-extrabold">{review.business_name}</h2>
            <p className="text-sm text-movo-slate">{review.branch_name}</p>
          </div>
          <button
            type="button"
            aria-label="Cerrar el detalle"
            onClick={onClose}
            className="grid size-10 shrink-0 place-items-center rounded-full bg-movo-sand outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
          >
            <XIcon />
          </button>
        </div>

        <div className="flex items-center justify-between gap-3">
          <span className="sr-only">{review.rating} de 5 estrellas</span>
          <span aria-hidden="true">
            <StarRating
              average={review.rating}
              count={MIN_REVIEWS_TO_RATE}
              starClassName="size-5"
              className="text-lg"
            />
          </span>
          <span className="text-sm text-movo-slate">{formatReviewDate(review.created_at)}</span>
        </div>

        <p className="text-sm font-semibold">{review.display_name || "Nombre borrado"}</p>

        <blockquote className="rounded-2xl bg-movo-cream p-4 whitespace-pre-line">
          {review.comment ? (
            review.comment
          ) : (
            <span className="text-movo-slate italic">
              {review.status === "hidden"
                ? "El texto se borró o la reseña no tenía comentario."
                : "La reseña no tiene comentario."}
            </span>
          )}
        </blockquote>

        {message ? <Notice title={message.title} body={message.body} /> : null}

        <div className="grid gap-3 sm:grid-cols-2">
          <button
            type="button"
            disabled={!allowed.visible || pending !== null}
            onClick={() => void decide("visible")}
            className="h-12 rounded-full bg-movo-card font-bold text-white outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-40"
          >
            {pending === "visible" ? "Guardando…" : "Dejar visible"}
          </button>
          <button
            type="button"
            disabled={!allowed.hidden || pending !== null}
            onClick={() => void decide("hidden")}
            className="h-12 rounded-full bg-movo-danger/10 font-bold text-movo-danger outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-40"
          >
            {pending === "hidden" ? "Guardando…" : "Ocultar reseña"}
          </button>
        </div>

        <p className="text-xs text-movo-slate">
          La reseña oculta deja de verse en la ficha del local y no cuenta en su promedio. A las 72
          horas se borran su texto y su nombre, y ya no se puede volver a dejar visible.
        </p>
      </aside>
    </>
  );
}
