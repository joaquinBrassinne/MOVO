"use client";

import type { SubmitEvent } from "react";

import { BusinessFields } from "@/components/business/BusinessFields";
import { CheckCircleIcon } from "@/components/socios/icons";
import type { Business } from "@/lib/business/types";
import { useBusinessForm } from "@/lib/business/useBusinessForm";

import { Shell, StepFooter } from "./Shell";

interface Props {
  business: Business | null;
  /** Se llama después de guardar, para volver a pedir los datos. */
  onSaved: () => void;
  onNext: () => void;
}

/** Paso 1: nombre, categoría, descripción y WhatsApp del negocio. */
export function StepBusiness({ business, onSaved, onNext }: Props) {
  const form = useBusinessForm(business);

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    if (await form.submit()) {
      onSaved();
      onNext();
    }
  }

  return (
    <Shell step={1}>
      <section className="rounded-3xl bg-white p-6 shadow-sm">
        <p className="inline-flex items-center gap-2 rounded-full bg-movo-accent/30 px-4 py-2 text-sm font-bold">
          <CheckCircleIcon className="size-4" />
          ¡Gracias por contratar MOVO!
        </p>
        <h2 className="mt-4 font-display text-3xl font-extrabold">Datos de tu negocio</h2>
        <p className="mt-2 text-base leading-relaxed text-movo-slate">
          Completá los datos de tu negocio. Confirmamos tu pago y revisamos tu perfil antes de
          publicarlo.
        </p>
      </section>

      <form
        id="step-form"
        onSubmit={submit}
        noValidate
        aria-busy={form.pending}
        className="space-y-6 rounded-3xl bg-white p-6 shadow-sm"
      >
        <BusinessFields form={form} />
      </form>

      <p className="px-2 text-center text-sm text-movo-slate">
        Verificamos tu perfil nosotros, después de aprobar tu negocio.
      </p>

      <StepFooter pending={form.pending} />
    </Shell>
  );
}
