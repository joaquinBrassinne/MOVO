"use client";

import Link from "next/link";

import { useMetric } from "@/lib/metrics/hooks";
import {
  direction,
  formatArs,
  formatChange,
  formatMultiple,
  formatPercent,
  formatPoints,
  peakCell,
  peakText,
  quotaCopy,
  returnFormula,
  returnMultiple,
  SLOT_KEYS,
  slotName,
  weekdayShort,
  heatLevel,
} from "@/lib/metrics/messages";
import type { Period, Rate } from "@/lib/metrics/types";

import { Bar, ChangeChip, MetricCard } from "./MetricCard";

function RateTile({ label, rate }: { label: string; rate: Rate | undefined }) {
  return (
    <div className="rounded-2xl bg-movo-cream p-4">
      <p className="text-xs font-semibold text-movo-slate">{label}</p>
      <p className="mt-1 font-display text-3xl font-extrabold">{formatPercent(rate?.rate)}</p>
      <p className="text-xs text-movo-slate">
        {rate ? `${rate.numerator} de ${rate.denominator}` : ""}
      </p>
      <div className="mt-2">
        <ChangeChip text={formatPoints(rate?.change)} tone={direction(rate?.change)} />
      </div>
    </div>
  );
}

export function ConversionCard({ period }: { period: Period }) {
  const { state, reload } = useMetric("conversion", period);

  return (
    <MetricCard
      title="Tasas de conversión"
      subtitle="De la atención digital al mostrador"
      state={state}
      onRetry={reload}
    >
      {(data) => (
        <div className="mt-4 grid grid-cols-2 gap-3">
          <RateTile label="Vistas → cupones" rate={data.views_to_coupons} />
          <RateTile label="Cupones → canjes" rate={data.coupons_to_redemptions} />
        </div>
      )}
    </MetricCard>
  );
}

/** An estimate, never a measurement: the card says so and shows the arithmetic behind it. */
export function EstimatedReturnCard({ period }: { period: Period }) {
  const { state, reload } = useMetric("estimated-return", period);

  return (
    <MetricCard title="Retorno estimado" state={state} onRetry={reload} dark badge="Estimado">
      {(data) => {
        if (data.status === "no_ticket") {
          return (
            <div className="mt-4 space-y-3">
              <p className="text-sm text-movo-muted">
                Para estimar el retorno de tus cupones, cargá el «Ticket promedio» de tu local.
              </p>
              <Link
                href="/owners/business/details"
                className="inline-flex h-10 items-center rounded-full bg-movo-accent px-5 text-sm font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-white"
              >
                Cargar ticket promedio en Datos del negocio
              </Link>
            </div>
          );
        }

        const estimated = data.estimated_return_ars ?? 0;
        const multiple =
          data.plan_price_ars === undefined
            ? null
            : returnMultiple(estimated, data.plan_price_ars, data.period.days);
        const movement = data.previous_estimated_return_ars
          ? (estimated - data.previous_estimated_return_ars) / data.previous_estimated_return_ars
          : null;

        return (
          <div className="mt-4">
            <p className="font-display text-4xl font-extrabold">
              ≈ {formatArs(estimated)}{" "}
              <span className="text-sm font-medium text-movo-muted">estimados</span>
            </p>
            <div className="mt-2 flex flex-wrap items-center gap-2">
              {multiple !== null ? (
                <span className="rounded-full bg-white/10 px-3 py-1 text-xs font-bold">
                  Retorno {formatMultiple(multiple)}
                </span>
              ) : null}
              {data.plan_price_ars ? (
                <span className="text-xs text-movo-muted">
                  Tu plan cuesta {formatArs(data.plan_price_ars)} por mes
                </span>
              ) : null}
              <ChangeChip text={formatChange(movement)} tone={direction(movement)} />
            </div>
            <div className="mt-4 rounded-2xl bg-white/10 p-4 text-sm">
              <p className="font-bold">Cómo se calcula</p>
              <p className="mt-1 text-movo-muted">
                {returnFormula(data.redemptions ?? 0, data.average_ticket_ars ?? 0)} ={" "}
                {formatArs(estimated)}.
              </p>
              <p className="mt-1 text-movo-muted">
                Es una estimación: supone que cada cupón canjeado gastó ese ticket. No mide ventas
                reales.
              </p>
            </div>
          </div>
        );
      }}
    </MetricCard>
  );
}

export function QuotaCard() {
  const { state, reload } = useMetric("quota-projection", 30);

  return (
    <MetricCard title="Cupo del mes" state={state} onRetry={reload} handlesStatus>
      {(data) => {
        const copy = quotaCopy(data);
        const used =
          data.limit && data.generated !== undefined ? data.generated / data.limit : null;
        return (
          <div className="mt-4 space-y-3">
            {used !== null && data.limit ? (
              <>
                <div className="flex items-baseline justify-between">
                  <p className="text-sm text-movo-slate">{formatPercent(used, 0)} consumido</p>
                  <p className="font-display font-bold">
                    {data.generated} / {data.limit}
                  </p>
                </div>
                <Bar fraction={used} className={used >= 1 ? "bg-movo-danger" : "bg-movo-page"} />
              </>
            ) : null}
            <p
              className={`rounded-2xl p-4 text-sm ${
                copy.tone === "warning"
                  ? "bg-movo-accent/30 text-movo-ink"
                  : "bg-movo-sand/70 text-movo-slate"
              }`}
            >
              {copy.text}
            </p>
          </div>
        );
      }}
    </MetricCard>
  );
}

const HEAT_SHADES = [
  "bg-movo-sand",
  "bg-movo-accent/25",
  "bg-movo-accent/50",
  "bg-movo-accent/75",
  "bg-movo-accent",
] as const;

export function HeatmapCard({ period }: { period: Period }) {
  const { state, reload } = useMetric("heatmap", period);

  return (
    <MetricCard
      title="Mapa de calor de canjes"
      subtitle="Cuándo se canjean tus cupones en sede"
      state={state}
      onRetry={reload}
    >
      {(data) => {
        const cells = data.cells ?? [];
        const max = Math.max(0, ...cells.map((cell) => cell.redemptions));
        const peak = peakCell(cells);
        const count = (weekday: number, slot: number) =>
          cells.find((cell) => cell.weekday === weekday && cell.slot === slot)?.redemptions ?? 0;

        return (
          <div className="mt-4">
            <div className="grid grid-cols-[3.5rem_repeat(7,minmax(0,1fr))] gap-1.5 text-center text-xs text-movo-slate">
              <span />
              {[1, 2, 3, 4, 5, 6, 7].map((weekday) => (
                <span key={weekday}>{weekdayShort(weekday)}</span>
              ))}
              {SLOT_KEYS.map((key, slot) => (
                <div key={key} className="contents">
                  <span className="self-center text-left">{slotName(key)}</span>
                  {[1, 2, 3, 4, 5, 6, 7].map((weekday) => {
                    const value = count(weekday, slot);
                    const isPeak = peak?.weekday === weekday && peak.slot === slot;
                    return (
                      <span
                        key={weekday}
                        title={`${weekdayShort(weekday)}, ${slotName(key).toLowerCase()}: ${value}`}
                        aria-label={`${weekdayShort(weekday)}, ${slotName(key).toLowerCase()}: ${value} canjes`}
                        className={`h-9 rounded-lg ${HEAT_SHADES[heatLevel(value, max)]} ${
                          isPeak ? "ring-2 ring-movo-ink" : ""
                        }`}
                      />
                    );
                  })}
                </div>
              ))}
            </div>
            <p className="mt-3 text-xs font-semibold text-movo-ink">{peakText(peak)}</p>
          </div>
        );
      }}
    </MetricCard>
  );
}
