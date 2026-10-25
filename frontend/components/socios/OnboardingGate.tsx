"use client";

import { useRouter } from "next/navigation";
import { useEffect, type ReactNode } from "react";

import { useBusiness } from "@/lib/business/BusinessProvider";
import { BUSINESS_STATUS } from "@/lib/business/types";
import { ONBOARDING_PATH, useRequireSession } from "@/lib/session/hooks";

/**
 * El panel es para negocios ya enviados. Quien todavía no creó el suyo, o lo dejó a
 * medias, vuelve al alta. Hasta que se sabe, no muestra nada.
 */
export function OnboardingGate({ children }: { children: ReactNode }) {
  const session = useRequireSession();
  const { business } = useBusiness();
  const router = useRouter();

  const needsOnboarding =
    business.status === "missing" ||
    (business.status === "ready" && business.data.business.status === BUSINESS_STATUS.DRAFT);

  useEffect(() => {
    if (needsOnboarding) router.replace(ONBOARDING_PATH);
  }, [needsOnboarding, router]);

  if (session.status !== "authenticated" || business.status === "loading" || needsOnboarding) {
    return null;
  }
  return children;
}
