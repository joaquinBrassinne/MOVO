"use client";

import { Notice } from "@/components/onboarding/fields";
import { AlertCircleIcon, InfoIcon } from "@/components/socios/icons";
import { useBusiness } from "@/lib/business/BusinessProvider";
import { useDowngradeAlerts, useSubscription } from "@/lib/subscription/hooks";
import { statusCopy } from "@/lib/subscription/messages";

import { PlanAlerts } from "./PlanAlerts";
import { PlanCard } from "./PlanCard";

/** Mi plan: the plan, what it includes and the state of the subscription. */
export function PlanView() {
  const { plan } = useBusiness();
  const { state: subscription, reload } = useSubscription();
  const { alerts, dismiss } = useDowngradeAlerts();

  if (plan.status === "loading" || subscription.status === "loading") {
    return (
      <p role="status" className="py-10 text-center text-movo-slate">
        Cargando tu plan…
      </p>
    );
  }

  if (plan.status === "failed" || subscription.status === "failed") {
    return (
      <div className="space-y-3">
        <Notice title="No pudimos cargar tu plan" body="Probá de nuevo en unos segundos." />
        <button
          type="button"
          onClick={() => (plan.status === "failed" ? window.location.reload() : reload())}
          className="h-11 w-full rounded-full bg-movo-sand font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
        >
          Reintentar
        </button>
      </div>
    );
  }

  if (plan.status === "missing" || subscription.status === "missing") {
    return (
      <p className="py-10 text-center text-movo-slate">
        Todavía no cargaste tu negocio. Tu plan aparece cuando lo completes.
      </p>
    );
  }

  const status = statusCopy(subscription.data);
  const attention = status.tone !== "calm";

  return (
    <div className="space-y-5">
      <PlanAlerts alerts={alerts} onDismiss={dismiss} />

      <PlanCard plan={plan.data} status={status} />

      {status.body ? (
        <section
          role={attention ? "alert" : undefined}
          className={`flex gap-3 rounded-3xl p-4 ${
            attention ? "border-l-4 border-movo-accent bg-white shadow-sm" : "bg-white/60"
          }`}
        >
          {attention ? (
            <AlertCircleIcon className="mt-0.5 size-5 shrink-0" />
          ) : (
            <InfoIcon className="mt-0.5 size-5 shrink-0 text-movo-slate" />
          )}
          <div>
            {status.title ? <h3 className="font-display font-bold">{status.title}</h3> : null}
            <p className="text-sm text-movo-slate">{status.body}</p>
          </div>
        </section>
      ) : null}

      <section className="rounded-3xl bg-white p-5 shadow-sm">
        <h3 className="font-display text-base font-bold">Gestión simple y transparente</h3>
        <p className="mt-2 text-sm text-movo-slate">
          Los cambios de plan y las bajas los gestiona a mano el equipo de MOVO.
        </p>
      </section>
    </div>
  );
}
