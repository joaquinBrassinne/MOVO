import type { Metadata } from "next";
import { Suspense } from "react";

import { ReviewFormView } from "@/components/reviews/form/ReviewFormView";

// El enlace lleva el token que abre el cupón: nunca se indexa ni se manda a otro sitio.
export const metadata: Metadata = {
  title: "Dejá tu reseña — MOVO",
  robots: { index: false, follow: false },
  referrer: "no-referrer",
};

export default function ReviewPage({ params }: PageProps<"/cupon/[token]/review">) {
  return (
    <Suspense fallback={null}>
      <ReviewFormView params={params} />
    </Suspense>
  );
}
