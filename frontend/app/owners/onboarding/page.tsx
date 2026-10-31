import type { Metadata } from "next";

import { OnboardingFlow } from "@/components/onboarding/OnboardingFlow";

export const metadata: Metadata = { title: "Alta de tu negocio — MOVO Socios" };

export default function OnboardingPage() {
  return <OnboardingFlow />;
}
