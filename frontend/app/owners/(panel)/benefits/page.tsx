import type { Metadata } from "next";

import { BenefitsView } from "@/components/benefits/BenefitsView";

export const metadata: Metadata = { title: "Beneficios — MOVO Socios" };

export default function BenefitsPage() {
  return <BenefitsView />;
}
