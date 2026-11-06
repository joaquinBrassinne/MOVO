import type { Metadata } from "next";

import { ReviewsPanel } from "@/components/panel/ReviewsPanel";

export const metadata: Metadata = { title: "Reseñas — MOVO Socios" };

export default function OwnerReviewsPage() {
  return <ReviewsPanel />;
}
