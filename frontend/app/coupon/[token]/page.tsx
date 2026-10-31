import type { Metadata } from "next";
import { Suspense } from "react";

import { CouponView } from "@/components/coupons/CouponView";

// El enlace lleva el token que abre el cupón: nunca se indexa ni se manda a otro sitio.
export const metadata: Metadata = {
  title: "Tu cupón — MOVO",
  robots: { index: false, follow: false },
  referrer: "no-referrer",
};

export default function CouponPage({ params }: PageProps<"/coupon/[token]">) {
  return (
    <Suspense fallback={null}>
      <CouponView params={params} />
    </Suspense>
  );
}
