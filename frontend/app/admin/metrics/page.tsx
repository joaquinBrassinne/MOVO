import type { Metadata } from "next";

import { MetricsView } from "@/components/admin/metrics/MetricsView";

export const metadata: Metadata = { title: "Métricas — MOVO Super admin" };

export default function MetricsPage() {
  return <MetricsView />;
}
