"use client";

import { useMetric } from "@/lib/metrics/hooks";
import { formatPercent } from "@/lib/metrics/messages";
import type { Essentials, Period } from "@/lib/metrics/types";

import { Bar, MetricCard } from "./MetricCard";

const NUMBER = new Intl.NumberFormat("es-AR");

/** Views of the profile and coupons generated and redeemed: counts, never people. */
export function EssentialCards({ period }: { period: Period }) {
  const { state, reload } = useMetric("essentials", period);

  const cards: { title: string; hint: string; value: (data: Essentials) => number }[] = [
    {
      title: "Vistas a tu perfil",
      hint: "Veces que se abrió la ficha de tu local",
      value: (data) => data.profile_views,
    },
    {
      title: "Cupones generados",
      hint: "Códigos solicitados por clientes potenciales",
      value: (data) => data.coupons.generated,
    },
    {
      title: "Cupones canjeados",
      hint: "Cupones que se presentaron en mostrador",
      value: (data) => data.coupons.redeemed,
    },
  ];

  return (
    <>
      {cards.map((card) => (
        <MetricCard key={card.title} title={card.title} state={state} onRetry={reload}>
          {(data) => (
            <>
              <p className="mt-3 font-display text-5xl font-extrabold">
                {NUMBER.format(card.value(data))}
              </p>
              <p className="mt-1 text-sm text-movo-slate">{card.hint}</p>
            </>
          )}
        </MetricCard>
      ))}
    </>
  );
}

/** The top services by views plus WhatsApp taps, each with its share of the ranking. */
export function TopActivities({ period }: { period: Period }) {
  const { state, reload } = useMetric("activities", period);

  return (
    <MetricCard
      title="Actividades más demandadas"
      subtitle="Interés medido por vistas y toques en WhatsApp"
      state={state}
      onRetry={reload}
    >
      {({ activities }) => {
        const demand = (a: { views: number; whatsapp_clicks: number }) =>
          a.views + a.whatsapp_clicks;
        const total = activities.reduce((sum, a) => sum + demand(a), 0);
        if (total === 0) {
          return (
            <p className="mt-4 rounded-2xl bg-movo-sand/70 p-4 text-sm text-movo-slate">
              Todavía nadie vio ni consultó tus actividades en este período.
            </p>
          );
        }
        return (
          <ol className="mt-4 space-y-4">
            {activities.map((activity, index) => (
              <li key={activity.service_id}>
                <div className="flex items-baseline justify-between gap-3">
                  <p className="font-display font-bold">
                    {index + 1} {activity.name}
                  </p>
                  <p className="shrink-0 text-xs text-movo-slate">
                    {NUMBER.format(activity.views)} vistas ·{" "}
                    {NUMBER.format(activity.whatsapp_clicks)} toques
                    <span className="ml-2 text-sm font-bold text-movo-ink">
                      {formatPercent(demand(activity) / total, 0)}
                    </span>
                  </p>
                </div>
                <div className="mt-1.5">
                  <Bar fraction={demand(activity) / total} />
                </div>
              </li>
            ))}
          </ol>
        );
      }}
    </MetricCard>
  );
}
