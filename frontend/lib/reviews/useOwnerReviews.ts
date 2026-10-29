"use client";

import { useCallback, useEffect, useState } from "react";

import type { FormMessage } from "@/lib/business/messages";
import { describeReportFailure, markReported } from "@/lib/reviews/ownerReviews";

import { listOwnerReviews, reportReview } from "./api";
import type { ModerationReview } from "./types";

const PAGE = 20;

export type OwnerReviewsState =
  | { status: "loading" }
  | { status: "failed" }
  | { status: "ready"; reviews: ModerationReview[]; nextCursor: string | null };

/**
 * The reviews of the owner's business, newest first, with cursor pagination and the report
 * action. A reported review stays in the list, marked, because it is still public.
 */
export function useOwnerReviews() {
  const [state, setState] = useState<OwnerReviewsState>({ status: "loading" });
  const [loadingMore, setLoadingMore] = useState(false);
  const [version, setVersion] = useState(0);

  useEffect(() => {
    let ignore = false;
    void listOwnerReviews({ limit: PAGE }).then((result) => {
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
  }, [version]);

  const reload = useCallback(() => {
    setState({ status: "loading" });
    setVersion((current) => current + 1);
  }, []);

  async function loadMore() {
    if (state.status !== "ready" || !state.nextCursor || loadingMore) return;
    setLoadingMore(true);
    const result = await listOwnerReviews({ limit: PAGE, cursor: state.nextCursor });
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

  /** Reports the review; returns the message to show when it failed, or `null`. */
  async function report(id: number): Promise<FormMessage | null> {
    const result = await reportReview(id);
    if (result.ok) {
      setState((current) =>
        current.status === "ready"
          ? { ...current, reviews: markReported(current.reviews, id) }
          : current,
      );
      return null;
    }
    if (result.error.code === "review_not_found") reload();
    return describeReportFailure(result.error);
  }

  return { state, loadingMore, loadMore, report, reload };
}
