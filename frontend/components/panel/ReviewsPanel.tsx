"use client";

import { useState } from "react";

import { Stars } from "@/components/reviews/list/Stars";
import { CheckCircleIcon, FlagIcon, StarIcon, UserIcon } from "@/components/socios/icons";
import type { FormMessage } from "@/lib/business/messages";
import { canReport } from "@/lib/reviews/ownerReviews";
import { relativeTime } from "@/lib/reviews/relativeTime";
import type { ModerationReview } from "@/lib/reviews/types";
import { useOwnerReviews } from "@/lib/reviews/useOwnerReviews";

import { ConfirmDialog } from "./ConfirmDialog";

const ANONYMOUS_NAME = "Cliente de MOVO";

/** Las reseñas del local para su dueño: puede reportar la que no cumpla las reglas. */
export function ReviewsPanel() {
  const { state, loadingMore, loadMore, report, reload } = useOwnerReviews();
  const [pending, setPending] = useState<ModerationReview | null>(null);
  const [failure, setFailure] = useState<FormMessage | null>(null);

  async function confirmReport() {
    if (!pending) return;
    const id = pending.id;
    setPending(null);
    setFailure(await report(id));
  }

  return (
    <div className="space-y-4">
      <p className="text-movo-slate">
        Lo que opinan los clientes que canjearon un cupón en tu local. Si una reseña no cumple las
        reglas, reportala: el equipo de MOVO la revisa y decide si se oculta.
      </p>

      {failure ? (
        <p role="alert" className="rounded-2xl bg-movo-danger/10 p-4 text-sm text-movo-ink">
          <strong className="block">{failure.title}</strong>
          {failure.body}
        </p>
      ) : null}

      {state.status === "loading" ? (
        <div aria-busy="true" aria-label="Cargando reseñas" className="space-y-3">
          {[0, 1, 2].map((n) => (
            <div key={n} className="h-32 animate-pulse rounded-3xl bg-white" />
          ))}
        </div>
      ) : null}

      {state.status === "failed" ? (
        <section role="alert" className="space-y-3 rounded-3xl bg-white p-6 text-center">
          <h2 className="font-display text-xl font-extrabold">No pudimos cargar las reseñas</h2>
          <p className="text-movo-slate">Revisá tu conexión y probá de nuevo.</p>
          <button
            type="button"
            onClick={reload}
            className="h-12 rounded-full bg-movo-accent px-6 font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
          >
            Reintentar
          </button>
        </section>
      ) : null}

      {state.status === "ready" && state.reviews.length === 0 ? (
        <section className="space-y-2 rounded-3xl bg-white p-6 text-center">
          <StarIcon className="mx-auto size-10 text-movo-gold" />
          <h2 className="font-display text-xl font-extrabold">Todavía no tenés reseñas</h2>
          <p className="text-movo-slate">
            Aparecen acá cuando un cliente deja la suya después de canjear un cupón en tu local.
          </p>
        </section>
      ) : null}

      {state.status === "ready" && state.reviews.length > 0 ? (
        <>
          <ul className="space-y-3">
            {state.reviews.map((review) => (
              <OwnerReviewCard
                key={review.id}
                review={review}
                onReport={() => {
                  setFailure(null);
                  setPending(review);
                }}
              />
            ))}
          </ul>
          {state.nextCursor ? (
            <button
              type="button"
              onClick={() => void loadMore()}
              disabled={loadingMore}
              className="h-12 w-full rounded-full bg-white font-bold shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-60"
            >
              {loadingMore ? "Cargando…" : "Ver más"}
            </button>
          ) : null}
        </>
      ) : null}

      <ConfirmDialog
        open={pending !== null}
        title="¿Reportar esta reseña?"
        body="Sigue visible para todos hasta que el equipo de MOVO la revise. Reportá solo las que incumplen las reglas: no se pueden ocultar por ser negativas."
        confirmLabel="Reportar"
        onConfirm={() => void confirmReport()}
        onCancel={() => setPending(null)}
      />
    </div>
  );
}

function OwnerReviewCard({ review, onReport }: { review: ModerationReview; onReport: () => void }) {
  const anonymous = review.display_name === ANONYMOUS_NAME;
  const initial = review.display_name.trim().charAt(0).toUpperCase();

  return (
    <li className="rounded-3xl bg-white p-4 shadow-sm">
      <div className="flex items-start gap-3">
        <span
          aria-hidden="true"
          className={`grid size-10 shrink-0 place-items-center rounded-full font-display font-bold ${
            anonymous ? "bg-movo-page text-movo-accent" : "bg-movo-sand text-movo-ink"
          }`}
        >
          {anonymous ? <UserIcon className="size-5" /> : initial}
        </span>
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-2">
            <h3 className="min-w-0 truncate font-display text-lg leading-tight font-bold">
              {review.display_name}
            </h3>
            <Stars
              value={review.rating}
              label={`${review.rating} de 5 estrellas`}
              className="size-5"
            />
          </div>
          <p className="text-sm text-movo-slate">
            {relativeTime(review.created_at)} · {review.branch_name}
          </p>
        </div>
      </div>

      {review.comment ? (
        <p className="mt-3 break-words whitespace-pre-line text-movo-ink">{review.comment}</p>
      ) : null}

      <div className="mt-3 flex items-center justify-between gap-3">
        <p className="inline-flex items-center gap-1.5 rounded-full bg-movo-accent/30 px-3 py-1 text-xs font-bold text-movo-gold">
          <CheckCircleIcon className="size-4" />
          Verificada por canje
        </p>
        {canReport(review) ? (
          <button
            type="button"
            onClick={onReport}
            className="inline-flex h-11 items-center gap-1.5 rounded-full px-3 text-sm font-bold text-movo-slate outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
          >
            <FlagIcon className="size-4" />
            Reportar
          </button>
        ) : (
          <p className="inline-flex items-center gap-1.5 rounded-full bg-movo-sand px-3 py-1 text-xs font-bold text-movo-ink">
            <FlagIcon className="size-4" />
            Reportada · en revisión
          </p>
        )}
      </div>
    </li>
  );
}
