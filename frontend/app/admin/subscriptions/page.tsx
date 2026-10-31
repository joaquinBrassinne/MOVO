import type { Metadata } from "next";

import { SubscriptionsView } from "@/components/admin/SubscriptionsView";

export const metadata: Metadata = { title: "Suscripciones — MOVO Super admin" };

export default function SubscriptionsPage() {
  return <SubscriptionsView />;
}
