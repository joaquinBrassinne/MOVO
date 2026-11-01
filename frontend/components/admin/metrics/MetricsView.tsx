"use client";

import { useState } from "react";

import { Notice } from "@/components/onboarding/fields";
import { couponsOf, formatCount, formatPercent, redemptionRate } from "@/lib/admin/metrics/format";
import { useAdminMetrics } from "@/lib/admin/metrics/hooks";
import type { MetricsPeriod } from "@/lib/admin/metrics/types";
import { formatDate } from "@/lib/admin/subscriptions/format";
import { useRequireSuperAdmin } from "@/lib/session/hooks";

import { AdminShell } from "../AdminShell";

const PERIODS: { value: MetricsPeriod; label: string }[] = [
  { value: "month", label: "Este mes" },
  { value: "all", label: "Acumulado" },
];

/** Métricas globales de la red: contadores agregados, sin datos personales. */
export function MetricsView() {
  const allowed = useRequireSuperAdmin();
  if (!allowed) return null;

  return (
    <AdminShell>
      <Metrics />
    </AdminShell>
  );
}

function Metrics() {
  const { state, reload } = useAdminMetrics();
  const [period, setPeriod] = useState<MetricsPeriod>("month");

  const metrics = state.status === "ready" ? state.data : null;
  const coupons = metrics ? couponsOf(metrics, period) : null;
  const scope = period === "month" ? "este mes" : "acumulados";

  return (
    <div className="space-y-6">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="font-display text-3xl font-extrabold">Métricas globales de la red</h1>
          <p className="mt-2 max-w-2xl text-movo-slate">
            Adopción comercial y efectividad de canjes. Son conteos de cupones, sin datos
            personales.
          </p>
        </div>
        <button
          type="button"
          onClick={reload}
          className="h-11 rounded-full bg-movo-accent px-5 text-sm font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
        >
          Actualizar datos
        </button>
      </header>

      <div role="tablist" aria-label="Período" className="flex gap-2">
        {PERIODS.map(({ value, label }) => (
          <button
            key={value}
            type="button"
            role="tab"
            aria-selected={value === period}
            onClick={() => setPeriod(value)}
            className={`h-10 rounded-full px-4 text-sm font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent ${
              value === period ? "bg-movo-card text-movo-accent" : "bg-movo-sand"
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      {state.status === "failed" ? (
        <Notice
          title="No pudimos cargar las métricas"
          body="Tocá «Actualizar datos» para volver a intentar."
        />
      ) : null}
      {state.status === "loading" ? (
        <p className="p-8 text-center text-movo-slate">Cargando…</p>
      ) : null}

      {metrics && coupons ? (
        <>
          <ul className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4" aria-label="Indicadores">
            <Card label="Locales activos" value={formatCount(metrics.active_businesses)} />
            <Card
              label={`Cupones generados ${scope}`}
              value={formatCount(coupons.generated)}
              hint={period === "month" ? `Desde el ${formatDate(metrics.month_start)}.` : undefined}
            />
            <Card
              label={`Cupones canjeados ${scope}`}
              value={formatCount(coupons.redeemed)}
              hint="Validados en el mostrador."
            />
            <Card
              label="Tasa de canje"
              value={formatPercent(redemptionRate(coupons))}
              hint="Canjeados sobre generados."
            />
          </ul>
          {coupons.generated === 0 ? (
            <p className="text-center text-movo-slate">
              {period === "month"
                ? "Todavía no se generaron cupones este mes."
                : "Todavía no se generaron cupones."}
            </p>
          ) : null}
        </>
      ) : null}
    </div>
  );
}

function Card({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <li className="rounded-3xl bg-white p-5 shadow-sm">
      <p className="text-xs font-bold tracking-wide text-movo-slate uppercase">{label}</p>
      <p className="mt-2 font-display text-4xl font-extrabold">{value}</p>
      {hint ? <p className="text-sm text-movo-slate">{hint}</p> : null}
    </li>
  );
}
