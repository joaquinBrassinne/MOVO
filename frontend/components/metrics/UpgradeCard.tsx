import Link from "next/link";

import { ArrowRightIcon } from "@/components/socios/icons";
import { formatArs } from "@/lib/metrics/messages";

/** The locked "Inteligencia comercial" card of a plan without the advanced metrics. */
export function UpgradeCard({ proPrice }: { proPrice: number | null }) {
  return (
    <section
      aria-labelledby="commercial-intelligence"
      className="rounded-3xl bg-movo-page p-6 text-white shadow-sm"
    >
      <div className="flex items-center gap-2">
        <h3 id="commercial-intelligence" className="font-display text-xl font-extrabold">
          Inteligencia comercial
        </h3>
        <span className="rounded-full bg-movo-accent px-2.5 py-0.5 text-xs font-bold text-movo-ink">
          PRO
        </span>
      </div>
      <p className="mt-1 text-xs text-movo-muted">Disponible en el Plan Pro</p>
      <p className="mt-4 text-sm leading-relaxed">
        Descubrí tasas de conversión completas, mapa de calor horario para optimizar tus turnos,
        retorno estimado en pesos y tus récords históricos para potenciar la ocupación de tu local.
      </p>
      <Link
        href="/owners/plan"
        className="mt-5 flex h-12 items-center justify-center gap-2 rounded-full bg-movo-accent font-display font-extrabold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-white"
      >
        {proPrice === null ? "Pasar a Pro" : `Pasar a Pro (${formatArs(proPrice)}/mes)`}
        <ArrowRightIcon className="size-5" />
      </Link>
      <p className="mt-3 text-center text-xs text-movo-muted">
        Activación asistida por el equipo de MOVO
      </p>
    </section>
  );
}
