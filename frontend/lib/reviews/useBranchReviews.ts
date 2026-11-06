"use client";

import { useEffect, useState } from "react";

import type { ApiError } from "@/lib/session/types";

import { getReviewSummary, listBranchReviews } from "./api";
import type { PublicReview, ReviewListFilters, ReviewSummary } from "./types";

/** The star chips of the public list: every review, exactly 5 stars, or 4 stars and up. */
export type ReviewFilter = "all" | "5" | "4plus";

const filterParams = (filter: ReviewFilter): ReviewListFilters =>
  filter === "5" ? { rating: 5 } : filter === "4plus" ? { min_rating: 4 } : {};

interface ListState {
  key: string;
  items: PublicReview[];
  nextCursor: string | null;
  error: ApiError | null;
  loadingMore: boolean;
  moreFailed: boolean;
}

interface SummaryState {
  key: string;
  summary: ReviewSummary | null;
  error: ApiError | null;
}

export interface BranchReviews {
  summary: { status: "loading" } | { status: "ready"; data: ReviewSummary } | { status: "failed" };
  filter: ReviewFilter;
  setFilter: (filter: ReviewFilter) => void;
  list:
    | { status: "loading" }
    | { status: "failed"; error: ApiError }
    | {
        status: "ready";
        items: PublicReview[];
        hasMore: boolean;
        loadingMore: boolean;
        moreFailed: boolean;
      };
  loadMore: () => void;
  retry: () => void;
}

/**
 * The reviews of one branch: the summary, plus the list for the chosen star chip with
 * cursor pagination. Changing the chip restarts the list; "load more" appends the next page.
 */
export function useBranchReviews(branchId: number): BranchReviews {
  const [filter, setFilter] = useState<ReviewFilter>("all");
  const [attempt, setAttempt] = useState(0);
  const [summaryState, setSummaryState] = useState<SummaryState | null>(null);
  const [listState, setListState] = useState<ListState | null>(null);

  const summaryKey = `${branchId}:${attempt}`;
  const listKey = `${branchId}:${filter}:${attempt}`;

  useEffect(() => {
    let ignore = false;
    void getReviewSummary(branchId).then((result) => {
      if (ignore) return;
      setSummaryState({
        key: summaryKey,
        summary: result.ok ? result.data : null,
        error: result.ok ? null : result.error,
      });
    });
    return () => {
      ignore = true;
    };
    // `summaryKey` resumes the branch and the retry counter.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [summaryKey]);

  useEffect(() => {
    let ignore = false;
    void listBranchReviews(branchId, filterParams(filter)).then((result) => {
      if (ignore) return;
      setListState({
        key: listKey,
        items: result.ok ? (result.data.reviews ?? []) : [],
        nextCursor: result.ok ? result.data.next_cursor : null,
        error: result.ok ? null : result.error,
        loadingMore: false,
        moreFailed: false,
      });
    });
    return () => {
      ignore = true;
    };
    // `listKey` resumes the branch, the chip and the retry counter: a new chip starts over.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [listKey]);

  const current = listState && listState.key === listKey ? listState : null;

  function loadMore() {
    if (!current || !current.nextCursor || current.loadingMore) return;
    const key = listKey;
    const cursor = current.nextCursor;
    setListState({ ...current, loadingMore: true, moreFailed: false });

    void listBranchReviews(branchId, { ...filterParams(filter), cursor }).then((result) => {
      // A chip change or a retry while this was in flight makes the page stale: drop it.
      setListState((state) => {
        if (!state || state.key !== key) return state;
        if (!result.ok) return { ...state, loadingMore: false, moreFailed: true };
        return {
          ...state,
          items: [...state.items, ...(result.data.reviews ?? [])],
          nextCursor: result.data.next_cursor,
          loadingMore: false,
          moreFailed: false,
        };
      });
    });
  }

  const currentSummary = summaryState && summaryState.key === summaryKey ? summaryState : null;

  return {
    summary: !currentSummary
      ? { status: "loading" }
      : currentSummary.summary
        ? { status: "ready", data: currentSummary.summary }
        : { status: "failed" },
    filter,
    setFilter,
    list: !current
      ? { status: "loading" }
      : current.error
        ? { status: "failed", error: current.error }
        : {
            status: "ready",
            items: current.items,
            hasMore: current.nextCursor !== null,
            loadingMore: current.loadingMore,
            moreFailed: current.moreFailed,
          },
    loadMore,
    retry: () => setAttempt((n) => n + 1),
  };
}
