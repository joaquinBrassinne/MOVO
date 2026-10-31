import { formatPrice, planFeatures } from "@/lib/plans/format";
import type { Plan } from "@/lib/plans/types";

const COPY = {
  starter: {
    name: "Starter",
    tag: "Plan inicial",
    blurb: "Ideal para estudios y gimnasios que recién empiezan en MOVO.",
  },
  pro: {
    name: "Pro",
    tag: "Máximo rendimiento",
    blurb: "Para clubes y negocios que buscan más alcance y mejor posicionamiento.",
  },
} as const;

export function PlanCard({ plan, featured }: { plan: Plan; featured: boolean }) {
  const copy = COPY[plan.plan];
  const headingId = `plan-${plan.plan}`;
  return (
    <li
      className={`relative flex flex-col rounded-3xl bg-white p-6 shadow-sm sm:p-8 ${
        featured ? "ring-2 ring-movo-accent" : "ring-1 ring-slate-200"
      }`}
    >
      {featured && (
        <span className="absolute -top-3 right-6 rounded-full bg-movo-accent px-3 py-1 text-xs font-bold text-movo-ink">
          ★ MÁS ELEGIDO
        </span>
      )}
      <p className="w-fit rounded-full bg-movo-sand px-2.5 py-1 text-xs font-semibold tracking-wide text-movo-slate uppercase">
        {copy.tag}
      </p>
      <h3 id={headingId} className="font-display mt-3 text-2xl font-bold text-movo-ink">
        {copy.name}
      </h3>
      <p className="mt-1 text-sm text-movo-slate">{copy.blurb}</p>
      <p className="mt-5 flex items-baseline gap-2">
        <span className="font-display text-4xl font-extrabold text-movo-ink">
          {formatPrice(plan.price)}
        </span>
        <span className="text-sm text-movo-slate">/ mes</span>
      </p>
      <p className="mt-1 text-xs text-movo-slate">Facturación mensual en pesos argentinos</p>
      <ul className="mt-6 flex-1 space-y-3 border-t border-slate-200 pt-6">
        {planFeatures(plan).map((feature) => (
          <li key={feature} className="flex gap-3 text-sm text-slate-800">
            <span aria-hidden="true" className="font-bold text-movo-gold">
              ✓
            </span>
            {feature}
          </li>
        ))}
      </ul>
      <a
        href={plan.checkout_url}
        rel="noopener"
        aria-describedby={headingId}
        className={`mt-8 block rounded-full px-6 py-3 text-center text-sm font-bold focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-movo-ink ${
          featured
            ? "bg-movo-accent text-movo-ink hover:brightness-95"
            : "bg-movo-page text-white hover:bg-movo-card"
        }`}
      >
        Contratar {copy.name} →
      </a>
    </li>
  );
}
