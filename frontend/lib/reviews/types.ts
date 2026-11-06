// Contrato de los endpoints de reseñas de la API Go.

/** Con menos reseñas que estas el local se muestra como "Nuevo" (SDD 5.5). */
export const MIN_REVIEWS_TO_RATE = 3;

export interface PublicReview {
  rating: number;
  comment?: string;
  display_name: string;
  created_at: string;
}

export interface ReviewPage {
  reviews: PublicReview[];
  next_cursor: string | null;
}

export interface ReviewListFilters {
  cursor?: string;
  limit?: number;
  /** Puntuación exacta (1 a 5). */
  rating?: number;
  /** Puntuación mínima (1 a 5). */
  min_rating?: number;
}

export type StarCount = "1" | "2" | "3" | "4" | "5";

export interface ReviewSummary {
  count: number;
  average: number;
  distribution: Record<StarCount, number>;
}

export type ReviewFormState = "open" | "reviewed" | "window_closed" | "not_redeemed";

export interface ReviewFormContext {
  state: ReviewFormState;
  business_name: string;
  branch_name: string | null;
  city: string | null;
  code: string;
  display_name: string;
  redeemed_at: string | null;
  review_until: string | null;
}

export interface ReviewInput {
  rating: number;
  comment: string;
  show_name: boolean;
}

export interface CreatedReview {
  rating: number;
  comment?: string;
  display_name: string;
  show_name: boolean;
  created_at: string;
}

export type ModerationStatus = "visible" | "reported" | "hidden";

export interface ModerationReview {
  id: number;
  branch_id: number;
  branch_name: string;
  business_id: number;
  business_name: string;
  rating: number;
  comment?: string;
  display_name: string;
  status: ModerationStatus;
  created_at: string;
}

export interface ModerationPage {
  reviews: ModerationReview[];
  next_cursor: string | null;
}

export interface ReviewStatus {
  id: number;
  status: ModerationStatus;
}

export type AdminQueue = "reported" | "hidden";
export type AdminDecision = "hidden" | "visible";
