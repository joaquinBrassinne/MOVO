import { formatPrice, planFeatures } from "@/lib/plans/format";
import type { Plan } from "@/lib/plans/types";

const COPY = {
  starter: {
    name: "Starter",
    blurb: "Ideal para estudios y gimnasios que recién empiezan en MOVO.",
  },
  pro: {
    name: "Pro",
    blurb: "Para clubes y negocios que buscan más alcance y mejor posicionamiento.",
  },
} as const;

function CheckIcon() {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 18 18"
      className="mt-px size-[18px] shrink-0 text-movo-accent"
    >
      <path
        d="M4 9.5l3 3 7-7"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

export function PlanCard({ plan, featured }: { plan: Plan; featured: boolean }) {
  const copy = COPY[plan.plan];
  const headingId = `plan-${plan.plan}`;
  return (
    // The featured card gets a gold gradient border: a 1.5px gradient frame around the dark card.
    <li
      className={`rounded-[28px] ${
        featured
          ? "bg-linear-160 from-movo-accent via-movo-accent/5 via-60% p-[1.5px]"
          : "bg-movo-muted/15 p-px"
      }`}
    >
      <div
        className={`relative flex h-full flex-col rounded-[27px] p-8 sm:p-9 ${
          featured ? "bg-movo-page" : "bg-movo-card"
        }`}
      >
        {featured && (
          <span className="absolute top-6 right-6 rounded-lg bg-movo-accent px-2.5 py-1.5 text-xs font-bold text-movo-ink">
            Más elegido
          </span>
        )}
        <h3 id={headingId} className="font-display text-2xl font-bold">
          {copy.name}
        </h3>
        <p className="mt-1.5 max-w-[30ch] text-sm leading-relaxed text-movo-muted">{copy.blurb}</p>
        <p className="mt-8 flex items-baseline gap-2">
          <span className="font-display text-6xl font-extrabold tracking-tight">
            {formatPrice(plan.price)}
          </span>
          <span className="text-movo-muted">/ mes</span>
        </p>
        <p className="mt-1 text-xs text-movo-muted">Facturación mensual en pesos argentinos</p>
        <a
          href={plan.checkout_url}
          rel="noopener"
          aria-describedby={headingId}
          className={`mt-7 block rounded-full px-6 py-4 text-center font-bold focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-movo-accent ${
            featured
              ? "bg-movo-accent text-movo-ink hover:brightness-95"
              : "bg-white/10 text-white ring-1 ring-movo-muted/25 hover:bg-white/15"
          }`}
        >
          Contratar {copy.name}
        </a>
        <ul className="mt-7 space-y-3">
          {planFeatures(plan).map((feature) => (
            <li key={feature} className="flex gap-2.5 leading-snug">
              <CheckIcon />
              {feature}
            </li>
          ))}
        </ul>
      </div>
    </li>
  );
}
