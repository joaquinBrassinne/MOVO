"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, type SubmitEvent } from "react";

import { BranchLocationFields } from "@/components/business/BranchLocationFields";
import { HoursEditor } from "@/components/business/HoursEditor";
import { Notice } from "@/components/onboarding/fields";
import { ArrowLeftIcon, CheckCircleIcon, ClockIcon } from "@/components/socios/icons";
import { setBranchHours } from "@/lib/business/api";
import { useBusiness } from "@/lib/business/BusinessProvider";
import { firstProblem, toDayHours, toSlots, type DayHours } from "@/lib/business/hours";
import { describeFailure } from "@/lib/business/messages";
import type { Branch } from "@/lib/business/types";
import { useBranchForm } from "@/lib/business/useBranchForm";

const LIST_PATH = "/socios/sedes";

/** Crea una sede (`branchId` nulo) o edita una existente. */
export function BranchEditor({ branchId }: { branchId: number | null }) {
  const { business } = useBusiness();
  if (business.status !== "ready") return null;

  const existing =
    branchId === null ? null : (business.data.branches.find((b) => b.id === branchId) ?? null);
  if (branchId !== null && existing === null) {
    return (
      <div className="space-y-4">
        <Notice title="No encontramos esa sede" body="Puede que ya no exista." />
        <BackLink />
      </div>
    );
  }

  return <BranchEditorForm key={existing?.id ?? "new"} existing={existing} />;
}

function BackLink() {
  return (
    <Link
      href={LIST_PATH}
      className="inline-flex items-center gap-2 text-sm font-bold text-movo-gold underline underline-offset-4"
    >
      <ArrowLeftIcon className="size-4" />
      Volver a Mis sedes
    </Link>
  );
}

function BranchEditorForm({ existing }: { existing: Branch | null }) {
  const router = useRouter();
  const { reload } = useBusiness();
  const form = useBranchForm(existing, "");
  const [days, setDays] = useState<DayHours[]>(() => toDayHours(existing?.hours ?? []));
  const [hoursError, setHoursError] = useState<string | null>(null);
  // La sede ya se creó pero sus horarios no se pudieron guardar: no se vuelve a crear.
  const [createdId, setCreatedId] = useState<number | null>(null);

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();

    const problem = firstProblem(days);
    if (problem) {
      setHoursError(
        problem.kind === "overlap"
          ? "Revisá los horarios: hay franjas que se pisan en un mismo día."
          : "Revisá los horarios: en cada franja, la hora de cierre va después de la de apertura.",
      );
      return;
    }
    setHoursError(null);

    const id = await form.submit();
    if (id === null) return;

    const saved = await setBranchHours(id, toSlots(days));
    if (!saved.ok) {
      form.setFeedback(describeFailure(saved));
      setCreatedId(id);
      reload();
      return;
    }

    reload();
    router.push(LIST_PATH);
  }

  const isNew = existing === null;

  return (
    <div className="space-y-5">
      <BackLink />

      <section>
        <p className="inline-flex rounded-full bg-movo-accent/30 px-4 py-2 text-xs font-bold tracking-wide uppercase">
          {isNew ? "Nueva sede" : "Editar sede"}
        </p>
        <h2 className="mt-3 font-display text-3xl font-extrabold">
          {isNew ? "Nueva sede" : existing.name}
        </h2>
        <p className="mt-2 text-base text-movo-slate">
          Sumá una ubicación física a tu cuenta para que los clientes te encuentren en el mapa de
          MOVO.
        </p>
      </section>

      <form
        id="branch-form"
        onSubmit={submit}
        noValidate
        aria-busy={form.pending || form.searching}
        className="space-y-5"
      >
        <BranchLocationFields form={form} askName />

        <section className="space-y-3 rounded-3xl bg-white p-5 shadow-sm">
          <div className="flex items-start justify-between gap-3">
            <div>
              <h3 className="flex items-center gap-2 font-display text-lg font-bold">
                <ClockIcon className="size-5 text-movo-gold" />
                Horarios de atención
              </h3>
              <p className="mt-1 text-sm text-movo-slate">
                Configurá los días y franjas en que está abierta esta sede.
              </p>
            </div>
          </div>

          <HoursEditor days={days} onChange={setDays} />

          {hoursError ? <Notice title="Horarios incorrectos" body={hoursError} /> : null}
        </section>

        {createdId === null ? (
          <button
            type="submit"
            disabled={form.pending || form.searching}
            className="flex h-14 w-full items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-60"
          >
            {form.pending ? "Guardando…" : "Guardar sede"}
            {form.pending ? null : <CheckCircleIcon />}
          </button>
        ) : (
          <div className="space-y-3">
            <p className="rounded-2xl bg-movo-sand p-4 text-sm">
              La sede ya se creó, pero sus horarios no se guardaron. Podés cargarlos editándola.
            </p>
            <Link
              href={`${LIST_PATH}/${createdId}`}
              className="flex h-14 items-center justify-center rounded-full bg-movo-accent font-display text-lg font-bold"
            >
              Editar la sede
            </Link>
          </div>
        )}

        <p className="text-center">
          <Link href={LIST_PATH} className="text-sm font-bold underline underline-offset-4">
            Cancelar y volver
          </Link>
        </p>
      </form>
    </div>
  );
}
