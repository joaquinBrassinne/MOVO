import type { Metadata } from "next";

import { PlanView } from "@/components/panel/PlanView";

export const metadata: Metadata = { title: "Mi plan — MOVO Socios" };

export default function PlanPage() {
  return <PlanView />;
}
