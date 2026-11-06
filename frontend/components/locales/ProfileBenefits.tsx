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
          <li
            key={benefit.id}
            className="rounded-3xl bg-gradient-to-br from-movo-accent to-amber-200 p-5 text-movo-ink shadow-md"
          >
            <span className="inline-flex rounded-full bg-movo-ink px-3 py-1 text-xs font-extrabold text-movo-accent">
              {benefit.short_label}
            </span>
            <h3 className="mt-2 font-display text-lg leading-tight font-bold">
              {benefitHeadline(benefit)}
            </h3>
            {benefit.conditions ? (
              <p className="mt-1 text-sm font-medium text-movo-ink/80">{benefit.conditions}</p>
            ) : null}
            <button
              type="button"
              onClick={() => setChosen(benefit)}
              className="mt-4 h-12 w-full cursor-pointer rounded-full bg-movo-ink font-display text-base font-bold text-white outline-none focus-visible:ring-2 focus-visible:ring-movo-ink focus-visible:ring-offset-2 focus-visible:ring-offset-amber-200"
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
