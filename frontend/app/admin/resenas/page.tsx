import type { Metadata } from "next";

import { ModerationView } from "@/components/admin/reviews/ModerationView";

export const metadata: Metadata = { title: "Reseñas — MOVO Super admin" };

export default function ReviewsPage() {
  return <ModerationView />;
}
