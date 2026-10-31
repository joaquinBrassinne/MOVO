import type { Metadata } from "next";

import { MyCoupons } from "@/components/coupons/MyCoupons";

export const metadata: Metadata = { title: "Mis cupones — MOVO", robots: { index: false } };

export default function CouponsPage() {
  return <MyCoupons />;
}
