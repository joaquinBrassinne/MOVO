"use client";

import { useEffect, useState } from "react";

import { clampComment, describeSubmitFailure } from "@/lib/reviews/reviewFormLogic";
import { isToken } from "@/lib/coupons/saved";

import { createReview, getReviewForm } from "./api";
import type { CreatedReview, ReviewFormContext, ReviewFormState } from "./types";

export type ReviewView =
  | { status: "loading" }
  | { status: "missing" }
  | { status: "failed" }
  | { status: "ready"; state: ReviewFormState; context: ReviewFormContext }
  | { status: "confirmed"; context: ReviewFormContext; review: CreatedReview };

/** Loads the review form of a coupon by its access token and submits the review. */
export function useReviewForm(token: string) {
  const [fetched, setFetched] = useState<ReviewView>({ status: "loading" });
  // A link that is not a token is a coupon that does not exist: it is never requested.
  const view: ReviewView = isToken(token) ? fetched : { status: "missing" };

  const [rating, setRatingValue] = useState(0);
  const [comment, setCommentValue] = useState("");
  const [showName, setShowName] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [errors, setErrors] = useState<{ rating?: string; comment?: string; form?: string }>({});

  useEffect(() => {
    let ignore = false;
    if (!isToken(token)) return;

    void getReviewForm(token).then((result) => {
      if (ignore) return;
      if (result.ok)
        setFetched({ status: "ready", state: result.data.state, context: result.data });
      else setFetched(result.status === 404 ? { status: "missing" } : { status: "failed" });
    });
    return () => {
      ignore = true;
    };
  }, [token]);

  function setRating(value: number) {
    setRatingValue(value);
    setErrors((current) => ({ ...current, rating: undefined }));
  }

  function setComment(text: string) {
    setCommentValue(clampComment(text));
    setErrors((current) => ({ ...current, comment: undefined }));
  }

  async function submit() {
    if (view.status !== "ready" || submitting) return;
    if (rating < 1 || rating > 5) {
      setErrors({ rating: "Elegí de 1 a 5 estrellas para publicar." });
      return;
    }

    setSubmitting(true);
    setErrors({});
    const result = await createReview(token, {
      rating,
      comment: comment.trim(),
      show_name: showName,
    });
    setSubmitting(false);

    if (result.ok) {
      setFetched({ status: "confirmed", context: view.context, review: result.data });
      return;
    }
    const failure = describeSubmitFailure(result.error);
    if (failure.kind === "state") setFetched({ ...view, state: failure.state });
    else setErrors({ [failure.kind === "generic" ? "form" : failure.kind]: failure.message });
  }

  return {
    view,
    rating,
    setRating,
    comment,
    setComment,
    showName,
    setShowName,
    submitting,
    errors,
    submit,
  };
}
