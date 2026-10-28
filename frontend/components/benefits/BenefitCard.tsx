import { Switch } from "@/components/panel/Switch";
import { PencilIcon } from "@/components/socios/icons";
import { benefitHeadline } from "@/lib/benefits/format";
import type { BenefitListItem } from "@/lib/benefits/types";
import { BENEFIT_TYPE } from "@/lib/benefits/types";

import { GiftIcon, TrendingUpIcon } from "./icons";

interface Props {
  benefit: BenefitListItem;
  /** El cupo del mes se agotó: el beneficio no entrega cupones aunque esté activo. */
  quotaExhausted: boolean;
  busy: boolean;
  onToggle: (next: boolean) => void;
  onEdit: () => void;
}

/** Una tarjeta del beneficio: su etiqueta, el título, las condiciones y lo que hizo este mes. */
export function BenefitCard({ benefit, quotaExhausted, busy, onToggle, onEdit }: Props) {
  const title = benefitHeadline(benefit);
  const paused = benefit.admin_paused;
  const stalled = !paused && benefit.active && quotaExhausted;
  const dimmed = paused || stalled || !benefit.active;

  return (
    <li
      className={`rounded-3xl bg-white p-5 shadow-sm ${dimmed ? "opacity-70" : ""}`}
      data-testid="benefit-card"
    >
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 flex-wrap items-center gap-2">
          <span className="inline-flex items-center gap-1.5 rounded-full bg-movo-accent px-3 py-1.5 text-sm font-extrabold">
            {benefit.type === BENEFIT_TYPE.CUSTOM ? <GiftIcon className="size-4" /> : null}
            {benefit.short_label}
          </span>
          {paused ? (
            <span className="rounded-full bg-movo-danger/10 px-3 py-1 text-xs font-bold text-movo-danger">
              Pausado por MOVO
            </span>
          ) : stalled ? (
            <span className="rounded-full bg-movo-sand px-3 py-1 text-xs font-bold text-movo-slate">
              Pausado
            </span>
          ) : null}
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <button
            type="button"
            aria-label={`Editar ${title}`}
            onClick={onEdit}
            className="grid size-10 place-items-center rounded-full bg-movo-sand outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
          >
            <PencilIcon className="size-4" />
          </button>
          <Switch
            checked={benefit.active}
            label={`${title}: ${benefit.active ? "activo" : "apagado"}`}
            disabled={busy || paused}
            onChange={onToggle}
          />
        </div>
      </div>

      <h3 className="mt-4 font-display text-xl leading-tight font-bold">{title}</h3>
      {benefit.conditions ? (
        <p className="mt-1 text-sm text-movo-slate">{benefit.conditions}</p>
      ) : null}
      {paused ? (
        <p className="mt-2 text-sm text-movo-danger">
          MOVO pausó este beneficio y no se ofrece a los clientes. Escribinos si creés que fue un
          error.
        </p>
      ) : null}

      <p className="mt-4 flex items-center gap-2 rounded-2xl bg-movo-cream px-4 py-3 text-sm font-semibold">
        <TrendingUpIcon className="size-4 shrink-0" />
        {benefit.generated_this_month}{" "}
        {benefit.generated_this_month === 1 ? "generado" : "generados"} ·{" "}
        {benefit.redeemed_this_month} {benefit.redeemed_this_month === 1 ? "canjeado" : "canjeados"}{" "}
        este mes
      </p>
    </li>
  );
}
