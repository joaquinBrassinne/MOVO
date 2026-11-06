import type { Metadata } from "next";
import { connection } from "next/server";
import { Suspense } from "react";

import { PlansLanding } from "@/components/plans/PlansLanding";
import { getPlans } from "@/lib/plans/api";
import type { Plan } from "@/lib/plans/types";

export const metadata: Metadata = {
  title: "Planes para tu negocio — MOVO",
  description: "Elegí el plan de MOVO para tu gimnasio, estudio o complejo deportivo.",
};

export default function PlansPage() {
  return (
    <Suspense fallback={null}>
      <PlansContent />
    </Suspense>
  );
}

/** Rendered per request: the build runs without the API, so the catalog cannot be read then. */
async function PlansContent() {
  await connection();

  let plans: Plan[] | null = null;
  try {
    plans = await getPlans();
  } catch {
    // Calm notice instead of the cards; the cause stays out of the page.
  }
  return <PlansLanding plans={plans} />;
}
