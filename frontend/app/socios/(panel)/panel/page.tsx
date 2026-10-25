import type { Metadata } from "next";

import { SessionStatus } from "@/components/socios/SessionStatus";

export const metadata: Metadata = { title: "Panel — MOVO Socios" };

// Provisorio: el inicio del panel se diseña con las fases de cupones y métricas. Existe
// para que el ingreso tenga adónde llegar y se pueda comprobar la sesión y el cierre.
export default function PanelPage() {
  return (
    <section className="rounded-[28px] bg-movo-card p-5 text-white">
      <SessionStatus />
    </section>
  );
}
