"use client";

import type { SubmitEvent } from "react";

import { BranchLocationFields } from "@/components/business/BranchLocationFields";
import { MapPinIcon } from "@/components/socios/icons";
import type { Branch } from "@/lib/business/types";
import { useBranchForm } from "@/lib/business/useBranchForm";

import { Shell, StepFooter } from "./Shell";

interface Props {
  /** La sede ya guardada, si el dueño vuelve a este paso. */
  branch: Branch | null;
  onSaved: () => void;
  onNext: () => void;
  onBack: () => void;
}

/** Paso 2: dirección de la sede, pin en el mapa y link de reservas. */
export function StepBranch({ branch, onSaved, onNext, onBack }: Props) {
  const form = useBranchForm(branch);

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    if (await form.submit()) {
      onSaved();
      onNext();
    }
  }

  return (
    <Shell step={2}>
      <section>
        <p className="inline-flex items-center gap-2 rounded-full bg-movo-accent/30 px-4 py-2 text-xs font-bold tracking-wide uppercase">
          <MapPinIcon className="size-4" />
          Paso 2 de 4 · Ubicación
        </p>
        <h2 className="mt-4 font-display text-4xl font-extrabold">Tu sede</h2>
        <p className="mt-2 text-base text-movo-slate">
          ¿Dónde te van a encontrar los clientes en el mapa de MOVO?
        </p>
      </section>

      <form
        id="step-form"
        onSubmit={submit}
        noValidate
        aria-busy={form.pending || form.searching}
        className="space-y-5"
      >
        <BranchLocationFields form={form} />
      </form>

      <StepFooter onBack={onBack} pending={form.pending} disabled={form.searching} />
    </Shell>
  );
}
