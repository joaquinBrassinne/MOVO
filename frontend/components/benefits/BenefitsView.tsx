"use client";

import { useState } from "react";

import { Notice } from "@/components/onboarding/fields";
import { InfoIcon, PlusIcon } from "@/components/socios/icons";
import { setBenefitActive } from "@/lib/benefits/api";
import { useBenefits, useCouponQuota } from "@/lib/benefits/hooks";
import { QUOTA_LEVEL, summarizeQuota } from "@/lib/benefits/quota";
import type { BenefitListItem } from "@/lib/benefits/types";
import { describeFailure, type FormMessage } from "@/lib/business/messages";

import { BenefitCard } from "./BenefitCard";
import { BenefitSheet } from "./BenefitSheet";
import { GiftIcon } from "./icons";
import { QuotaCard } from "./QuotaCard";
import { QuotaSheet } from "./QuotaSheet";

/** Lo que está abierto en una hoja inferior: nada, un beneficio nuevo, uno que se edita o el cupo. */
type Sheet =
  { kind: "none" } | { kind: "benefit"; benefit: BenefitListItem | null } | { kind: "quota" };

/** Beneficios y cupo mensual: la tarjeta del cupo, la lista de beneficios y el alta. */
export function BenefitsView() {
  const benefits = useBenefits();
  const quota = useCouponQuota();
  const [sheet, setSheet] = useState<Sheet>({ kind: "none" });
  const [message, setMessage] = useState<FormMessage | null>(null);
  const [busyId, setBusyId] = useState<number | null>(null);

  const list = benefits.state.status === "ready" ? benefits.state.data : [];
  const exhausted =
    quota.state.status === "ready" &&
    summarizeQuota(quota.state.data).level === QUOTA_LEVEL.EXHAUSTED;

  function reload() {
    benefits.reload();
    quota.reload();
  }

  async function toggle(benefit: BenefitListItem, active: boolean) {
    setBusyId(benefit.id);
    setMessage(null);

    const result = await setBenefitActive(benefit.id, active);
    if (!result.ok) setMessage(describeFailure(result).form ?? null);

    setBusyId(null);
    benefits.reload();
  }

  return (
    <div className="space-y-5">
      <div>
        <h2 className="font-display text-3xl leading-tight font-extrabold">
          Beneficios de tu local
        </h2>
        <p className="mt-1 text-base text-movo-slate">
          Gestioná las promociones que ven los alumnos en MOVO.
        </p>
      </div>

      {quota.state.status === "ready" ? (
        <QuotaCard quota={quota.state.data} onEdit={() => setSheet({ kind: "quota" })} />
      ) : quota.state.status === "loading" ? (
        <div
          aria-busy="true"
          aria-label="Cargando el cupo del mes"
          className="h-56 animate-pulse rounded-3xl bg-white"
        />
      ) : (
        <Notice title="No pudimos cargar tu cupo del mes" body="Probá de nuevo en unos minutos." />
      )}

      <p className="flex items-start gap-3 rounded-3xl bg-movo-page p-4 text-sm text-white">
        <InfoIcon className="mt-0.5 size-5 shrink-0 text-movo-accent" />
        Cada cliente puede tener un cupón activo a la vez. Cuando lo usa o vence, puede generar
        otro.
      </p>

      {message ? <Notice title={message.title} body={message.body} /> : null}

      {benefits.state.status === "loading" ? (
        <ul aria-busy="true" aria-label="Cargando tus beneficios" className="space-y-3">
          {[0, 1].map((key) => (
            <li key={key} className="h-44 animate-pulse rounded-3xl bg-white" />
          ))}
        </ul>
      ) : null}

      {benefits.state.status === "failed" ? (
        <div className="space-y-3">
          <Notice
            title="No pudimos cargar tus beneficios"
            body="Revisá tu conexión y probá de nuevo."
          />
          <button
            type="button"
            onClick={benefits.reload}
            className="h-12 w-full rounded-full bg-movo-sand font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
          >
            Reintentar
          </button>
        </div>
      ) : null}

      {benefits.state.status === "ready" && list.length === 0 ? (
        <section className="rounded-3xl bg-white p-6 text-center shadow-sm">
          <span className="mx-auto grid size-20 place-items-center rounded-full bg-movo-accent/30">
            <GiftIcon className="size-10" />
          </span>
          <h3 className="mt-4 font-display text-2xl font-extrabold">
            Todavía no creaste beneficios
          </h3>
          <p className="mt-2 text-movo-slate">
            Un buen beneficio hace que más gente te elija. Atraé alumnos con un descuento en su
            primera clase o un regalo de bienvenida.
          </p>
          <button
            type="button"
            onClick={() => setSheet({ kind: "benefit", benefit: null })}
            className="mt-5 flex h-14 w-full items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
          >
            <PlusIcon />
            Crear mi primer beneficio
          </button>
        </section>
      ) : null}

      {list.length > 0 ? (
        <>
          <div className="flex items-baseline justify-between">
            <h3 className="font-display text-xl font-bold">
              Beneficios <span className="text-movo-slate">({list.length})</span>
            </h3>
          </div>
          <ul className="space-y-3">
            {list.map((benefit) => (
              <BenefitCard
                key={benefit.id}
                benefit={benefit}
                quotaExhausted={exhausted}
                busy={busyId === benefit.id}
                onToggle={(next) => void toggle(benefit, next)}
                onEdit={() => setSheet({ kind: "benefit", benefit })}
              />
            ))}
          </ul>
          <button
            type="button"
            onClick={() => setSheet({ kind: "benefit", benefit: null })}
            className="flex h-14 w-full items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
          >
            <PlusIcon />
            Nuevo beneficio
          </button>
        </>
      ) : null}

      <p className="text-center text-sm text-movo-slate">
        Los cambios se ven en la app apenas los guardás.
      </p>

      {sheet.kind === "benefit" ? (
        <BenefitSheet
          // La clave hace que cada beneficio abra con su propio formulario limpio.
          key={sheet.benefit?.id ?? "new"}
          benefit={sheet.benefit}
          onClose={() => setSheet({ kind: "none" })}
          onSaved={reload}
        />
      ) : null}
      {sheet.kind === "quota" && quota.state.status === "ready" ? (
        <QuotaSheet
          quota={quota.state.data}
          onClose={() => setSheet({ kind: "none" })}
          onSaved={reload}
        />
      ) : null}
    </div>
  );
}
