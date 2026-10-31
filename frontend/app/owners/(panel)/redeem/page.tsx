import type { Metadata } from "next";

import { RedeemView } from "@/components/redeem/RedeemView";

export const metadata: Metadata = { title: "Canjear cupón — MOVO Socios" };

export default function RedeemPage() {
  return <RedeemView />;
}
