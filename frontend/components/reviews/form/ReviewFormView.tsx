"use client";

import { use } from "react";

import { ReviewFormScreen } from "./ReviewFormScreen";

/** Reads the coupon token from the route. */
export function ReviewFormView({ params }: { params: Promise<{ token: string }> }) {
  const { token } = use(params);
  return <ReviewFormScreen token={token} />;
}
