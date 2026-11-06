"use client";

import { useState } from "react";

import { Notice } from "@/components/onboarding/fields";
import { StarRating } from "@/components/reviews/StarRating";
import { useRequireSuperAdmin } from "@/lib/session/hooks";
import { MIN_REVIEWS_TO_RATE, type AdminQueue } from "@/lib/reviews/types";
import { useModeration } from "@/lib/reviews/useModeration";

import { AdminShell } from "../AdminShell";
import { formatReviewDate, truncateComment } from "@/lib/reviews/moderation";
import { ReviewDetail } from "./ReviewDetail";

const TABS: { label: string; queue: AdminQueue }[] = [
  { label: "Reportadas", queue: "reported" },
  { label: "Ocultas", queue: "hidden" },
];

/** Moderación de reseñas: las reportadas y las ocultas, con el detalle de la elegida. */
export function ModerationView() {
  const allowed = useRequireSuperAdmin();
  if (!allowed) return null;

  return (
    <AdminShell>
      <Moderation />
    </AdminShell>
  );
}

function Moderation() {
  const [queue, setQueue] = useState<AdminQueue>("reported");

  return (
    <div className="space-y-6">
      <header>
        <h1 className="font-display text-3xl font-extrabold">Moderación de reseñas</h1>
        <p className="mt-2 max-w-2xl text-movo-slate">
          Revisá las reseñas que reportaron los locales y decidí si quedan visibles o se ocultan.
        </p>
      </header>

      <div role="tablist" aria-label="Cola de moderación" className="flex gap-2">
        {TABS.map((tab) => {
          const active = tab.queue === queue;
          return (
            <button
              key={tab.queue}
              type="button"
              role="tab"
              aria-selected={active}
              onClick={() => setQueue(tab.queue)}
              className={`h-10 rounded-full px-5 text-sm font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent ${
                active ? "bg-movo-card text-movo-accent" : "bg-movo-sand"
              }`}
            >
              {tab.label}
            </button>
          );
        })}
      </div>

      <Queue key={queue} queue={queue} />
    </div>
  );
}

function Queue({ queue }: { queue: AdminQueue }) {
  const { state, loadingMore, loadMore, decide, reload } = useModeration(queue);
  const [selectedId, setSelectedId] = useState<number | null>(null);

  const reviews = state.status === "ready" ? state.reviews : [];
  // The row leaves the list after a decision, so the panel closes with it.
  const selected = reviews.find((review) => review.id === selectedId) ?? null;

  return (
    <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_26rem]">
      <section className="overflow-hidden rounded-3xl bg-white shadow-sm">
        {state.status === "failed" ? (
          <div className="space-y-3 p-5">
            <Notice
              title="No pudimos cargar las reseñas"
              body="Revisá tu conexión y volvé a intentar."
            />
            <button type="button" onClick={reload} className="text-sm font-bold underline">
              Reintentar
            </button>
          </div>
        ) : null}
        {state.status === "loading" ? (
          <p className="p-8 text-center text-movo-slate">Cargando…</p>
        ) : null}

        {state.status === "ready" && reviews.length === 0 ? (
          <p className="p-8 text-center text-movo-slate">
            {queue === "reported" ? "No hay reseñas reportadas." : "No hay reseñas ocultas."}
          </p>
        ) : null}

        {reviews.length > 0 ? (
          <table className="w-full text-left text-sm">
            <thead className="bg-movo-sand text-xs font-bold tracking-wide text-movo-slate uppercase">
              <tr>
                <th scope="col" className="px-5 py-3">
                  Local y sede
                </th>
                <th scope="col" className="px-3 py-3">
                  Puntaje
                </th>
                <th scope="col" className="hidden px-3 py-3 md:table-cell">
                  Comentario
                </th>
                <th scope="col" className="hidden px-5 py-3 md:table-cell">
                  Fecha
                </th>
              </tr>
            </thead>
            <tbody>
              {reviews.map((review) => (
                <tr
                  key={review.id}
                  className={`border-t border-movo-sand ${review.id === selectedId ? "bg-movo-accent/15" : ""}`}
                >
                  <td className="px-5 py-4">
                    <button
                      type="button"
                      onClick={() => setSelectedId(review.id)}
                      aria-pressed={review.id === selectedId}
                      className="text-left outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
                    >
                      <span className="block font-display text-base font-bold">
                        {review.business_name}
                      </span>
                      <span className="block text-xs text-movo-slate">{review.branch_name}</span>
                    </button>
                  </td>
                  <td className="px-3 py-4">
                    <span className="sr-only">{review.rating} de 5 estrellas</span>
                    <span aria-hidden="true">
                      <StarRating average={review.rating} count={MIN_REVIEWS_TO_RATE} />
                    </span>
                  </td>
                  <td className="hidden max-w-xs px-3 py-4 text-movo-slate md:table-cell">
                    {review.comment ? (
                      truncateComment(review.comment)
                    ) : (
                      <span className="italic">Sin texto</span>
                    )}
                  </td>
                  <td className="hidden px-5 py-4 whitespace-nowrap md:table-cell">
                    {formatReviewDate(review.created_at)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : null}

        {state.status === "ready" && state.nextCursor ? (
          <div className="border-t border-movo-sand p-4 text-center">
            <button
              type="button"
              disabled={loadingMore}
              onClick={() => void loadMore()}
              className="h-11 rounded-full bg-movo-sand px-6 text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-50"
            >
              {loadingMore ? "Cargando…" : "Ver más"}
            </button>
          </div>
        ) : null}
      </section>

      {selected ? (
        <ReviewDetail
          // Each review opens with its own message and loading state.
          key={selected.id}
          review={selected}
          onClose={() => setSelectedId(null)}
          onDecide={decide}
        />
      ) : null}
    </div>
  );
}
