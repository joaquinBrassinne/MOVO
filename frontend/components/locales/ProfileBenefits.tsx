"use client";

import { useState } from "react";

import { GetCouponSheet } from "@/components/coupons/GetCouponSheet";
import { benefitHeadline } from "@/lib/benefits/format";
import type { ClaimedBenefit } from "@/lib/benefits/types";

interface Props {
  businessName: string;
  benefits: ClaimedBenefit[];
}

/** Los beneficios que el local ofrece hoy, cada uno con su botón para obtener el cupón. */
export function ProfileBenefits({ businessName, benefits }: Props) {
  const [chosen, setChosen] = useState<ClaimedBenefit | null>(null);

  if (benefits.length === 0) return null;

  return (
    <section aria-labelledby="benefits" className="mt-8">
      <div className="flex items-baseline justify-between">
        <h2 id="benefits" className="font-display text-xl font-bold">
          Beneficios
        </h2>
        <span className="text-sm font-semibold text-movo-slate">
          {benefits.length} {benefits.length === 1 ? "disponible" : "disponibles"}
        </span>
      </div>

      <ul className="mt-3 space-y-3">
        {benefits.map((benefit) => (
          <li key={benefit.id} className="rounded-3xl bg-white p-4 shadow-sm">
            <span className="inline-flex rounded-full bg-movo-accent px-3 py-1 text-xs font-extrabold">
              {benefit.short_label}
            </span>
            <h3 className="mt-2 font-display text-lg leading-tight font-bold">
              {benefitHeadline(benefit)}
            </h3>
            {benefit.conditions ? (
              <p className="mt-1 text-sm text-movo-slate">{benefit.conditions}</p>
            ) : null}
            <button
              type="button"
              onClick={() => setChosen(benefit)}
              className="mt-3 h-12 w-full rounded-full bg-movo-accent font-display text-base font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
            >
              Obtener cupón
            </button>
          </li>
        ))}
      </ul>

      {chosen ? (
        <GetCouponSheet
          // Cada beneficio abre su propio formulario limpio.
          key={chosen.id}
          businessName={businessName}
          benefit={chosen}
          onClose={() => setChosen(null)}
        />
      ) : null}
    </section>
  );
}
