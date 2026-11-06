import { formatRating, isNewBranch } from "@/lib/reviews/format";
import { MIN_REVIEWS_TO_RATE, type ReviewSummary, type StarCount } from "@/lib/reviews/types";

import { Stars } from "./Stars";

const STARS: StarCount[] = ["5", "4", "3", "2", "1"];

/**
 * The summary of a branch: average, count and the 5-to-1 distribution. With fewer than 3
 * reviews it shows the "Nuevo" treatment instead of an average (SDD 5.5).
 */
export function ReviewSummaryCard({ summary }: { summary: ReviewSummary }) {
  const isNew = isNewBranch(summary.count);

  return (
    <section aria-label="Resumen de reseñas" className="rounded-3xl bg-white p-5 shadow-sm">
      <div className="flex items-center gap-5">
        <div className="shrink-0 text-center">
          {isNew ? (
            <span className="inline-flex rounded-full bg-movo-accent/30 px-3 py-1.5 font-display text-lg font-extrabold text-movo-gold">
              Nuevo
            </span>
          ) : (
            <>
              <p className="font-display text-5xl leading-none font-extrabold">
                {formatRating(summary.average)}
              </p>
              <div className="mt-2">
                <Stars
                  value={summary.average}
                  label={`${formatRating(summary.average)} de 5 estrellas`}
                  className="size-4"
                />
              </div>
            </>
          )}
          <p className="mt-2 text-sm text-movo-slate">
            {summary.count} {summary.count === 1 ? "reseña real" : "reseñas reales"}
          </p>
        </div>

        {isNew ? null : <Distribution summary={summary} />}
      </div>

      {isNew ? (
        <p className="mt-4 text-sm text-movo-slate">
          Un local muestra su puntuación desde las {MIN_REVIEWS_TO_RATE} reseñas.
        </p>
      ) : null}

      <p className="mt-4 rounded-2xl bg-movo-sand px-3 py-2 text-sm text-movo-ink">
        Solo opinan <strong>personas que entrenaron con cupón presencial</strong>
      </p>
    </section>
  );
}

function Distribution({ summary }: { summary: ReviewSummary }) {
  return (
    <ul className="min-w-0 flex-1 space-y-1.5" aria-label="Reseñas por cantidad de estrellas">
      {STARS.map((star) => {
        const count = summary.distribution[star] ?? 0;
        const share = summary.count > 0 ? (count / summary.count) * 100 : 0;
        return (
          <li key={star} className="flex items-center gap-2 text-sm">
            <span className="w-3 text-movo-slate">{star}</span>
            <span
              aria-hidden="true"
              className="h-2 flex-1 overflow-hidden rounded-full bg-movo-sand"
            >
              <span
                className="block h-full rounded-full bg-movo-accent"
                style={{ width: `${share}%` }}
              />
            </span>
            <span className="w-8 text-right text-movo-slate">{count}</span>
          </li>
        );
      })}
    </ul>
  );
}
