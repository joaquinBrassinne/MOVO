"use client";

import { useBranchReviews } from "@/lib/reviews/useBranchReviews";

import { ReviewCard } from "./ReviewCard";
import { ReviewFilterChips } from "./ReviewFilterChips";
import { ReviewSummaryCard } from "./ReviewSummaryCard";

const RETRY_BUTTON =
  "h-11 rounded-full bg-movo-accent px-5 text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink";

/** The summary, the star chips and the paginated list of one branch. State lives in the hook. */
export function BranchReviewsView({ branchId }: { branchId: number }) {
  const { summary, filter, setFilter, list, loadMore, retry } = useBranchReviews(branchId);

  const total = summary.status === "ready" ? summary.data.count : 0;
  const empty = summary.status === "ready" && total === 0;

  return (
    <div className="space-y-5">
      {summary.status === "loading" ? (
        <div
          aria-busy="true"
          aria-label="Cargando el resumen"
          className="h-44 animate-pulse rounded-3xl bg-movo-sand"
        />
      ) : summary.status === "ready" ? (
        <ReviewSummaryCard summary={summary.data} />
      ) : null}

      {empty ? (
        <div className="rounded-3xl bg-white p-6 text-center shadow-sm">
          <h2 className="font-display text-xl font-bold">Todavía no hay reseñas</h2>
          <p className="mt-2 text-movo-slate">
            Cuando alguien canjee un cupón en este local, podrá dejar su opinión acá.
          </p>
        </div>
      ) : (
        <>
          {summary.status === "ready" ? (
            <ReviewFilterChips filter={filter} total={total} onChange={setFilter} />
          ) : null}

          {list.status === "loading" ? (
            <div aria-busy="true" aria-label="Cargando las reseñas" className="space-y-3">
              {[0, 1, 2].map((n) => (
                <div key={n} className="h-36 animate-pulse rounded-3xl bg-movo-sand" />
              ))}
            </div>
          ) : list.status === "failed" ? (
            <div role="alert" className="rounded-3xl bg-white p-6 text-center shadow-sm">
              <p className="font-semibold">No pudimos cargar las reseñas.</p>
              <button type="button" onClick={retry} className={`mt-3 ${RETRY_BUTTON}`}>
                Reintentar
              </button>
            </div>
          ) : list.items.length === 0 ? (
            <p className="rounded-3xl bg-white p-6 text-center text-movo-slate shadow-sm">
              No hay reseñas con ese filtro.
            </p>
          ) : (
            <>
              <ul className="space-y-3">
                {list.items.map((review, index) => (
                  <ReviewCard key={`${review.created_at}-${index}`} review={review} />
                ))}
              </ul>
              {list.moreFailed ? (
                <p role="alert" className="text-center text-sm text-movo-danger">
                  No pudimos traer más reseñas.
                </p>
              ) : null}
              {list.hasMore ? (
                <div className="text-center">
                  <button
                    type="button"
                    onClick={loadMore}
                    disabled={list.loadingMore}
                    className="h-12 rounded-full bg-white px-6 font-bold shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-60"
                  >
                    {list.loadingMore ? "Cargando…" : "Ver más"}
                  </button>
                </div>
              ) : null}
            </>
          )}
        </>
      )}

      {summary.status === "failed" ? (
        <div role="alert" className="rounded-3xl bg-white p-6 text-center shadow-sm">
          <p className="font-semibold">No pudimos cargar el resumen.</p>
          <button type="button" onClick={retry} className={`mt-3 ${RETRY_BUTTON}`}>
            Reintentar
          </button>
        </div>
      ) : null}
    </div>
  );
}
