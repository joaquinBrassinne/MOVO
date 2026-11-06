import { CheckCircleIcon, UserIcon } from "@/components/socios/icons";
import { relativeTime } from "@/lib/reviews/relativeTime";
import type { PublicReview } from "@/lib/reviews/types";

import { Stars } from "./Stars";

/** Reviews published without a name (SDD 5.5) show a person icon instead of an initial. */
const ANONYMOUS_NAME = "Cliente de MOVO";

/** One public review: who, when, stars, comment and the "verified by redemption" badge. */
export function ReviewCard({ review }: { review: PublicReview }) {
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
          <p className="text-sm text-movo-slate">{relativeTime(review.created_at)}</p>
        </div>
      </div>

      {review.comment ? (
        <p className="mt-3 break-words whitespace-pre-line text-movo-ink">{review.comment}</p>
      ) : null}

      <p className="mt-3 inline-flex items-center gap-1.5 rounded-full bg-movo-accent/30 px-3 py-1 text-xs font-bold text-movo-gold">
        <CheckCircleIcon className="size-4" />
        Reseña verificada por canje
      </p>
    </li>
  );
}
