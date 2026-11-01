"use client";

import { useState } from "react";

import { useAdvancedMetrics, useProPrice } from "@/lib/metrics/hooks";
import { DEFAULT_PERIOD, type Period } from "@/lib/metrics/types";

import {
  BenefitsCard,
  BranchesCard,
  RecordsCard,
  ReturningCard,
  ReviewsCard,
  TimeToRedeemCard,
} from "./CustomerCards";
import { EssentialCards, TopActivities } from "./EssentialCards";
import { PeriodTabs } from "./PeriodTabs";
import { ConversionCard, EstimatedReturnCard, HeatmapCard, QuotaCard } from "./ReturnCards";
import { UpgradeCard } from "./UpgradeCard";

/** Pro cards in the order of the design; each one loads, fails and empties on its own. */
function ProSections({ period }: { period: Period }) {
  return (
    <>
      <ConversionCard period={period} />
      <EstimatedReturnCard period={period} />
      <QuotaCard />
      <HeatmapCard period={period} />
      <BenefitsCard period={period} />
      <TimeToRedeemCard period={period} />
      <ReturningCard period={period} />
      <BranchesCard period={period} />
      <ReviewsCard period={period} />
      <RecordsCard />
    </>
  );
}

/** Métricas: the essentials for every plan, and the Pro sections when the plan includes them. */
export function MetricsView() {
  const [period, setPeriod] = useState<Period>(DEFAULT_PERIOD);
  const access = useAdvancedMetrics();
  const locked = access.status === "ready" && !access.data;
  const proPrice = useProPrice(locked);

  return (
    <div className="space-y-5">
      <div>
        <h2 className="font-display text-3xl leading-tight font-extrabold">
          {access.status === "ready" && access.data
            ? "Inteligencia comercial"
            : "Métricas esenciales"}
        </h2>
        <p className="mt-1 text-base text-movo-slate">
          El pulso de tu espacio en MOVO: vistas y cupones, sin datos personales.
        </p>
      </div>

      <PeriodTabs value={period} onChange={setPeriod} />

      <EssentialCards period={period} />
      <TopActivities period={period} />

      {access.status === "loading" ? (
        <div
          role="status"
          aria-label="Cargando"
          className="h-40 animate-pulse rounded-3xl bg-movo-sand"
        />
      ) : null}
      {access.status === "ready" ? (
        access.data ? (
          <ProSections period={period} />
        ) : (
          <UpgradeCard proPrice={proPrice} />
        )
      ) : null}
    </div>
  );
}
