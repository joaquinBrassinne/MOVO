import { CheckIcon, ClockIcon } from "@/components/socios/icons";
import type { PlanSummary } from "@/lib/business/types";
import { planFeatures, planLabel } from "@/lib/subscription/format";
import type { StatusCopy } from "@/lib/subscription/messages";

const CHIP_TONE = {
  calm: "bg-movo-card text-white",
  warning: "bg-movo-accent text-movo-ink",
  danger: "bg-movo-danger text-white",
} as const;

/** The dark card with the plan, its price, the validity chip and what it includes. */
export function PlanCard({ plan, status }: { plan: PlanSummary; status: StatusCopy }) {
  return (
    <section
      aria-labelledby="plan-name"
      className="rounded-3xl bg-movo-page p-6 text-white shadow-sm"
    >
      <div className="flex items-baseline justify-between gap-3">
        <h2 id="plan-name" className="font-display text-3xl font-extrabold">
          Plan {planLabel(plan.plan)}
        </h2>
        <p className="font-display text-xl font-extrabold text-movo-accent">
          ${plan.price.toLocaleString("es-AR")}
          <span className="text-sm font-medium text-movo-muted">/mes</span>
        </p>
      </div>

      <p
        role="status"
        className={`mt-4 inline-flex items-center gap-2 rounded-full px-3 py-1.5 text-sm font-semibold ${CHIP_TONE[status.tone]}`}
      >
        <ClockIcon className="size-4" />
        {status.chip}
      </p>

      <h3 className="mt-6 border-t border-movo-card pt-5 text-sm font-extrabold tracking-wide uppercase">
        Qué incluye tu plan
      </h3>
      <ul className="mt-3 space-y-3">
        {planFeatures(plan.plan, plan).map((feature) => (
          <li key={feature} className="flex items-start gap-3 text-sm">
            <span className="mt-0.5 grid size-5 shrink-0 place-items-center rounded-full bg-movo-accent/30 text-movo-accent">
              <CheckIcon className="size-3" />
            </span>
            {feature}
          </li>
        ))}
      </ul>
    </section>
  );
}
