"use client";

import { useMetric } from "@/lib/metrics/hooks";
import {
  direction,
  formatChange,
  formatPercent,
  formatPoints,
  formatStars,
  formatStarsChange,
  monthLabel,
  recordProgressText,
  recordsInsufficientText,
  slotName,
  slotRange,
  weekdayLong,
} from "@/lib/metrics/messages";
import type { Period } from "@/lib/metrics/types";

import { Bar, ChangeChip, MetricCard } from "./MetricCard";

export function BenefitsCard({ period }: { period: Period }) {
  const { state, reload } = useMetric("benefits", period);

  return (
    <MetricCard
      title="Rendimiento por beneficio"
      subtitle="Efectividad de cada propuesta"
      state={state}
      onRetry={reload}
    >
      {({ benefits = [] }) =>
        benefits.length === 0 ? (
          <p className="mt-4 text-sm text-movo-slate">Ningún beneficio generó cupones.</p>
        ) : (
          <ul className="mt-4 space-y-4">
            {benefits.map((benefit) => (
              <li key={benefit.benefit_id}>
                <div className="flex items-baseline justify-between gap-3">
                  <p className="font-display font-bold">{benefit.name}</p>
                  <p className="shrink-0 text-sm font-bold">
                    {formatPercent(benefit.rate)} <span className="font-medium">canje</span>
                  </p>
                </div>
                <div className="my-1.5">
                  <Bar fraction={benefit.rate ?? 0} />
                </div>
                <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-movo-slate">
                  <p>
                    {benefit.generated} generados · {benefit.redeemed} canjeados · {benefit.expired}{" "}
                    vencidos · {benefit.pending} pendientes
                  </p>
                  <ChangeChip
                    text={formatPoints(benefit.change)}
                    tone={direction(benefit.change)}
                  />
                </div>
              </li>
            ))}
          </ul>
        )
      }
    </MetricCard>
  );
}

export function TimeToRedeemCard({ period }: { period: Period }) {
  const { state, reload } = useMetric("time-to-redeem", period);

  return (
    <MetricCard
      title="Tiempo hasta el canje"
      subtitle="Velocidad de decisión de los alumnos"
      state={state}
      onRetry={reload}
    >
      {(data) => {
        const buckets = [
          { label: "Mismo día", rate: data.same_day, colour: "bg-movo-accent" },
          { label: "Hasta 24 h", rate: data.within_24h, colour: "bg-movo-page" },
          { label: "Hasta 48 h", rate: data.within_48h, colour: "bg-movo-slate/40" },
        ];
        return (
          <div className="mt-4">
            <div
              className="flex h-3 w-full overflow-hidden rounded-full bg-movo-sand"
              aria-hidden="true"
            >
              {buckets.map((bucket) => (
                <div
                  key={bucket.label}
                  className={bucket.colour}
                  style={{ width: `${(bucket.rate?.rate ?? 0) * 100}%` }}
                />
              ))}
            </div>
            <ul className="mt-3 grid grid-cols-3 gap-2 text-xs">
              {buckets.map((bucket) => (
                <li key={bucket.label}>
                  <span className="flex items-center gap-1.5 text-movo-slate">
                    <span className={`size-2 rounded-full ${bucket.colour}`} />
                    {bucket.label}
                  </span>
                  <span className="font-display text-xl font-extrabold">
                    {formatPercent(bucket.rate?.rate, 0)}
                  </span>
                </li>
              ))}
            </ul>
          </div>
        );
      }}
    </MetricCard>
  );
}

export function ReturningCard({ period }: { period: Period }) {
  const { state, reload } = useMetric("returning-customers", period);

  return (
    <MetricCard
      title="Clientes que volvieron"
      subtitle="Medido por repetición de canjes"
      state={state}
      onRetry={reload}
    >
      {(data) => (
        <div className="mt-4 grid grid-cols-2 gap-3">
          <div className="rounded-2xl bg-movo-cream p-4">
            <p className="text-xs font-semibold text-movo-slate">1 canje</p>
            <p className="font-display text-3xl font-extrabold">{data.one_redemption ?? 0}</p>
            <p className="text-xs text-movo-slate">Primera visita</p>
          </div>
          <div className="rounded-2xl bg-movo-cream p-4">
            <p className="text-xs font-semibold text-movo-slate">2 o más</p>
            <p className="font-display text-3xl font-extrabold">{data.two_or_more ?? 0}</p>
            <p className="text-xs text-movo-slate">Recurrencia activa</p>
            <div className="mt-2">
              <ChangeChip
                text={formatPoints(data.repeat_rate?.change)}
                tone={direction(data.repeat_rate?.change)}
              />
            </div>
          </div>
        </div>
      )}
    </MetricCard>
  );
}

export function BranchesCard({ period }: { period: Period }) {
  const { state, reload } = useMetric("branches", period);

  return (
    <MetricCard
      title="Comparativa por sede"
      subtitle="Atracción por punto comercial"
      state={state}
      onRetry={reload}
    >
      {({ branches = [] }) => (
        <ul className="mt-4 space-y-3">
          {branches.map((branch) => {
            const figures = [
              { label: "Vistas", count: branch.service_views },
              { label: "WhatsApp", count: branch.whatsapp_clicks },
              { label: "Canjes", count: branch.redemptions },
            ];
            return (
              <li key={branch.branch_id} className="rounded-2xl bg-movo-cream p-4">
                <div className="flex items-center justify-between gap-2">
                  <p className="font-display font-bold">{branch.name}</p>
                  {!branch.active ? (
                    <span className="rounded-full bg-movo-sand px-2.5 py-1 text-xs font-bold text-movo-slate">
                      Retirada
                    </span>
                  ) : null}
                </div>
                <dl className="mt-2 grid grid-cols-3 gap-2">
                  {figures.map(({ label, count }) => (
                    <div key={label}>
                      <dt className="text-xs text-movo-slate">{label}</dt>
                      <dd className="font-display text-2xl font-extrabold">{count.value}</dd>
                      <dd className="text-xs text-movo-slate">
                        {formatChange(count.change) ?? ""}
                      </dd>
                    </div>
                  ))}
                </dl>
              </li>
            );
          })}
        </ul>
      )}
    </MetricCard>
  );
}

export function ReviewsCard({ period }: { period: Period }) {
  const { state, reload } = useMetric("reviews", period);

  return (
    <MetricCard
      title="Reseñas en el tiempo"
      subtitle="Satisfacción agregada de asistentes"
      state={state}
      onRetry={reload}
    >
      {({ rating, review_rate: reviewRate }) => (
        <div className="mt-4 space-y-3">
          {rating?.average == null ? (
            <p className="text-sm text-movo-slate">Todavía no hay reseñas en este período.</p>
          ) : (
            <div className="flex items-center gap-3">
              <p className="font-display text-4xl font-extrabold">
                {formatStars(rating.average)} <span aria-label="estrellas">★</span>
              </p>
              <div>
                <p className="text-xs text-movo-slate">
                  Promedio de {rating.reviews} {rating.reviews === 1 ? "reseña" : "reseñas"}
                </p>
                {rating.change != null ? (
                  <ChangeChip
                    text={`${formatStarsChange(rating.change)} ★`}
                    tone={direction(rating.change)}
                  />
                ) : null}
              </div>
            </div>
          )}
          {reviewRate?.rate != null ? (
            <p className="rounded-2xl bg-movo-sand/70 p-4 text-sm text-movo-slate">
              El {formatPercent(reviewRate.rate, 0)} de los canjes deja reseña.
            </p>
          ) : null}
        </div>
      )}
    </MetricCard>
  );
}

export function RecordsCard() {
  const { state, reload } = useMetric("records", 30);

  return (
    <MetricCard
      title="Tus récords"
      subtitle="Hitos de tu espacio"
      state={state}
      onRetry={reload}
      handlesStatus
    >
      {(data) => {
        if (data.status === "insufficient_data" || !data.best_month) {
          return (
            <p className="mt-4 rounded-2xl bg-movo-sand/70 p-4 text-sm text-movo-slate">
              {recordsInsufficientText(data)}
            </p>
          );
        }
        const current = data.current_month;
        return (
          <div className="mt-4 space-y-4">
            <div>
              <p className="text-xs font-semibold text-movo-slate">Mejor mes histórico</p>
              <p className="font-display text-xl font-extrabold">
                {monthLabel(data.best_month.month)} · {data.best_month.redemptions} canjes
              </p>
              {current ? (
                <div className="mt-2 space-y-1.5">
                  <Bar fraction={current.progress} className="bg-movo-page" />
                  <p className="text-xs text-movo-slate">
                    <span className="font-bold text-movo-ink">{recordProgressText(current)}</span>
                    {" · "}
                    {current.redemptions} / {data.best_month.redemptions} canjes en{" "}
                    {monthLabel(current.month)}
                  </p>
                </div>
              ) : null}
            </div>
            <dl className="grid grid-cols-2 gap-3">
              <div className="rounded-2xl bg-movo-cream p-4">
                <dt className="text-xs text-movo-slate">Tu día fuerte</dt>
                <dd className="font-display font-bold">
                  {data.strong_weekday ? weekdayLong(data.strong_weekday) : "—"}
                  {data.strong_slot ? `, ${slotRange(data.strong_slot.key)}` : ""}
                </dd>
                {data.strong_slot ? (
                  <dd className="text-xs text-movo-slate">{slotName(data.strong_slot.key)}</dd>
                ) : null}
              </div>
              <div className="rounded-2xl bg-movo-cream p-4">
                <dt className="text-xs text-movo-slate">Beneficio estrella</dt>
                <dd className="font-display font-bold">{data.star_benefit?.name ?? "—"}</dd>
              </div>
            </dl>
          </div>
        );
      }}
    </MetricCard>
  );
}
