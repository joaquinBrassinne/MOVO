// Cliente tipado de los endpoints de reseñas. Cada función es una llamada a la API; el
// estado y los efectos viven en los hooks. Los errores llegan como `ApiResult` con el
// sobre `{error:{code,message,details}}` ya leído.

import { authed } from "@/lib/api/authed";
import { requestAnonymous, requestPublic } from "@/lib/api/request";

import type {
  AdminDecision,
  AdminQueue,
  CreatedReview,
  ModerationPage,
  ReviewFormContext,
  ReviewInput,
  ReviewListFilters,
  ReviewPage,
  ReviewStatus,
  ReviewSummary,
} from "./types";

function query(params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== "") search.set(key, String(value));
  }
  const text = search.toString();
  return text ? `?${text}` : "";
}

export const listBranchReviews = (branchId: number, filters: ReviewListFilters = {}) =>
  requestPublic<ReviewPage>(`/branches/${branchId}/reviews${query({ ...filters })}`);

export const getReviewSummary = (branchId: number) =>
  requestPublic<ReviewSummary>(`/branches/${branchId}/reviews/summary`);

export const getReviewForm = (token: string) =>
  requestAnonymous<ReviewFormContext>("GET", `/access/${encodeURIComponent(token)}/review`);

export const createReview = (token: string, input: ReviewInput) =>
  requestAnonymous<CreatedReview>("POST", `/access/${encodeURIComponent(token)}/reviews`, input);

export const listOwnerReviews = (filters: { cursor?: string; limit?: number } = {}) =>
  authed<ModerationPage>("GET", `/me/business/reviews${query({ ...filters })}`);

export const reportReview = (id: number) =>
  authed<ReviewStatus>("POST", `/me/business/reviews/${id}/report`);

export const listAdminReviews = (
  filters: { status?: AdminQueue; cursor?: string; limit?: number } = {},
) => authed<ModerationPage>("GET", `/admin/reviews${query({ ...filters })}`);

export const setReviewStatus = (id: number, status: AdminDecision) =>
  authed<ReviewStatus>("PUT", `/admin/reviews/${id}/status`, { status });
