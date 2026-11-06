"use client";

import { useCallback, useEffect, useState } from "react";

import { describeModerationFailure, withoutReview } from "@/lib/reviews/moderation";
import type { FormMessage } from "@/lib/business/messages";

import { listAdminReviews, setReviewStatus } from "./api";
import type { AdminDecision, AdminQueue, ModerationReview } from "./types";

const PAGE = 20;

export type ModerationState =
  | { status: "loading" }
  | { status: "failed" }
  | { status: "ready"; reviews: ModerationReview[]; nextCursor: string | null };

/**
 * One moderation queue ("reported" or "hidden") with cursor pagination and the two
 * decisions. A decided review leaves the list. Mount it with `key={queue}` so a tab
 * change starts clean.
 */
export function useModeration(queue: AdminQueue) {
  const [state, setState] = useState<ModerationState>({ status: "loading" });
  const [loadingMore, setLoadingMore] = useState(false);
  const [version, setVersion] = useState(0);

  useEffect(() => {
    let ignore = false;
    void listAdminReviews({ status: queue, limit: PAGE }).then((result) => {
      if (ignore) return;
      setState(
        result.ok
          ? { status: "ready", reviews: result.data.reviews, nextCursor: result.data.next_cursor }
          : { status: "failed" },
      );
    });
    return () => {
      ignore = true;
    };
  }, [queue, version]);

  const reload = useCallback(() => setVersion((current) => current + 1), []);

  async function loadMore() {
    if (state.status !== "ready" || !state.nextCursor || loadingMore) return;
    setLoadingMore(true);
    const result = await listAdminReviews({ status: queue, limit: PAGE, cursor: state.nextCursor });
    setLoadingMore(false);
    if (!result.ok) return;
    setState((current) =>
      current.status === "ready"
        ? {
            status: "ready",
            reviews: [
              ...current.reviews,
              ...result.data.reviews.filter((r) => !current.reviews.some((c) => c.id === r.id)),
            ],
            nextCursor: result.data.next_cursor,
          }
        : current,
    );
  }

  /** Applies the decision; returns the message to show when it failed, or `null`. */
  async function decide(id: number, decision: AdminDecision): Promise<FormMessage | null> {
    const result = await setReviewStatus(id, decision);
    if (result.ok) {
      setState((current) =>
        current.status === "ready"
          ? { ...current, reviews: withoutReview(current.reviews, id) }
          : current,
      );
      return null;
    }
    // Someone else may have decided it first: show the real queue again.
    if (result.error.code !== "review_purged") reload();
    return describeModerationFailure(result.error);
  }

  return { state, loadingMore, loadMore, decide, reload };
}
