"use client";

import { useState } from "react";

import { GetCouponSheet } from "@/components/coupons/GetCouponSheet";
import { ArrowRightIcon } from "@/components/socios/icons";
import { benefitHeadline } from "@/lib/benefits/format";
import type { ClaimedBenefit } from "@/lib/benefits/types";

import { BUTTON_FOCUS, HEADING, SECTION } from "./profileStyles";

interface Props {
  businessName: string;
  benefits: ClaimedBenefit[];
}

/** Los beneficios que el local ofrece hoy, como tickets: el descuento a la izquierda y el botón para obtener el cupón. */
export function ProfileBenefits({ businessName, benefits }: Props) {
  const [chosen, setChosen] = useState<ClaimedBenefit | null>(null);

  if (benefits.length === 0) return null;

  return (
    <section id="section-benefits" aria-labelledby="benefits" className={SECTION}>
      <div className="flex items-baseline justify-between">
        <h2 id="benefits" className={HEADING}>
          Beneficios · {benefits.length}
        </h2>
      </div>

      <ul className="mt-3.5 space-y-3">
        {benefits.map((benefit) => (
          <li
            key={benefit.id}
            className="flex items-stretch overflow-hidden rounded-2xl bg-movo-card"
          >
            <span className="grid min-w-[5.75rem] shrink-0 place-items-center border-r-2 border-dashed border-movo-tabs bg-movo-accent px-3 text-center font-display text-xl leading-tight font-extrabold tracking-tight whitespace-nowrap text-movo-ink">
              {benefit.short_label}
            </span>
            <div className="min-w-0 flex-1 px-4 py-3.5">
              <h3 className="font-display text-base leading-snug font-bold">
                {benefitHeadline(benefit)}
              </h3>
              {benefit.conditions ? (
                <p className="mt-0.5 text-[0.8125rem] text-movo-muted">{benefit.conditions}</p>
              ) : null}
              <button
                type="button"
                onClick={() => setChosen(benefit)}
                className={`mt-2 inline-flex min-h-9 cursor-pointer items-center gap-1 rounded text-[0.8125rem] font-bold text-movo-accent ${BUTTON_FOCUS}`}
              >
                Obtener cupón
                <ArrowRightIcon className="size-3.5" />
              </button>
            </div>
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
