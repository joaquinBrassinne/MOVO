import type { Metadata } from "next";

import { OnboardingGate } from "@/components/socios/OnboardingGate";
import { SessionStatus } from "@/components/socios/SessionStatus";

export const metadata: Metadata = { title: "Panel — MOVO Socios" };

// Provisorio: la Fase 2 lo reemplaza por el panel real del negocio. Existe para que
// el ingreso tenga adónde llegar y se pueda comprobar la sesión y el cierre.
export default function PanelPage() {
  return (
    <OnboardingGate>
      <section className="rounded-[28px] bg-movo-card p-5">
        <SessionStatus />
      </section>
    </OnboardingGate>
  );
}
