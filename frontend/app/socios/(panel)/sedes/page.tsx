import type { Metadata } from "next";

import { BranchesPanel } from "@/components/panel/BranchesPanel";

export const metadata: Metadata = { title: "Mis sedes — MOVO Socios" };

export default function BranchesPage() {
  return <BranchesPanel />;
}
