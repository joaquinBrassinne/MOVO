import { InfoIcon } from "@/components/socios/icons";
import { PLAN, type PlanSummary } from "@/lib/business/types";

interface Props {
  plan: PlanSummary;
  /** Cuántos tiene en uso y el tope del plan (`null` si no tiene). */
  used: number;
  limit: number | null;
  /** Qué se cuenta, en plural: "servicios", "sedes". */
  noun: string;
  /** Cómo se llama lo que se cuenta cuando está en uso: "activos", "activas". */
  inUse: string;
}

/** La tarjeta oscura con el plan y cuánto de su límite se usa. */
export function PlanUsage({ plan, used, limit, noun, inUse }: Props) {
  const percent = limit === null ? 0 : Math.min(100, Math.round((used / limit) * 100));
  const left = limit === null ? null : Math.max(0, limit - used);

  return (
    <section className="rounded-3xl bg-movo-page p-6 text-white shadow-sm">
      <p className="inline-flex items-center gap-2 rounded-full bg-movo-accent/30 px-3 py-1 text-xs font-bold tracking-wide text-movo-accent uppercase">
        <span className="size-2 rounded-full bg-movo-accent" aria-hidden="true" />
        Plan {plan.plan === PLAN.PRO ? "Pro" : "Starter"} activo
      </p>

      <div className="mt-5 flex items-baseline justify-between gap-4">
        <p className="font-display text-2xl font-extrabold">
          {limit === null ? `${used} ${noun} ${inUse}` : `${used} de ${limit} ${noun} ${inUse}`}
        </p>
        {limit === null ? null : (
          <p className="font-display text-xl font-extrabold text-movo-accent">{percent}%</p>
        )}
      </div>

      {limit === null ? null : (
        <div
          role="progressbar"
          aria-valuenow={used}
          aria-valuemin={0}
          aria-valuemax={limit}
          aria-label={`${noun} ${inUse}`}
          className="mt-3 h-2 overflow-hidden rounded-full bg-movo-card"
        >
          <div className="h-full rounded-full bg-movo-accent" style={{ width: `${percent}%` }} />
        </div>
      )}

      <p className="mt-4 flex items-center gap-2 text-sm text-movo-muted">
        <InfoIcon className="size-4 shrink-0" />
        {limit === null
          ? "Tu plan no tiene tope."
          : left === 0
            ? "Llegaste al límite de tu plan."
            : `Te ${left === 1 ? "queda 1 lugar disponible" : `quedan ${left} lugares disponibles`} en tu plan.`}
      </p>
    </section>
  );
}
