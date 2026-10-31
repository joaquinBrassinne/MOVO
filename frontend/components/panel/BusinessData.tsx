"use client";

import Link from "next/link";
import { useState, type SubmitEvent } from "react";

import { BusinessFields } from "@/components/business/BusinessFields";
import { ArrowLeftIcon, CheckCircleIcon } from "@/components/socios/icons";
import { useBusiness } from "@/lib/business/BusinessProvider";
import type { Business } from "@/lib/business/types";
import { useBusinessForm } from "@/lib/business/useBusinessForm";

/** Datos del negocio: edita nombre, categoría, descripción y WhatsApp; cada cambio queda registrado. */
export function BusinessData() {
  const { business } = useBusiness();

  return (
    <div className="space-y-5">
      <Link
        href="/owners/business"
        className="inline-flex items-center gap-2 text-sm font-bold text-movo-slate outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
      >
        <ArrowLeftIcon className="size-4" />
        Volver a Mi local
      </Link>
      <div>
        <h2 className="font-display text-3xl leading-tight font-extrabold">Datos del negocio</h2>
        <p className="mt-1 text-base text-movo-slate">
          Lo que ven los clientes en tu ficha. El link de tu ficha no cambia aunque cambies el
          nombre.
        </p>
      </div>

      {business.status === "ready" ? (
        // El formulario arranca con lo guardado; después de guardar ya muestra lo escrito.
        <BusinessDataForm key={business.data.business.id} existing={business.data.business} />
      ) : business.status === "loading" ? (
        <div
          aria-busy="true"
          aria-label="Cargando tus datos"
          className="h-96 animate-pulse rounded-3xl bg-white"
        />
      ) : (
        <p className="rounded-3xl bg-white p-6 text-movo-slate shadow-sm">
          No pudimos cargar los datos de tu negocio. Recargá la página para volver a intentar.
        </p>
      )}
    </div>
  );
}

function BusinessDataForm({ existing }: { existing: Business }) {
  const { reload } = useBusiness();
  const form = useBusinessForm(existing);
  const [saved, setSaved] = useState(false);

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaved(false);
    if (await form.submit()) {
      setSaved(true);
      reload();
    }
  }

  return (
    <form
      onSubmit={submit}
      noValidate
      aria-busy={form.pending}
      className="space-y-6 rounded-3xl bg-white p-6 shadow-sm"
    >
      <BusinessFields form={form} />

      {saved ? (
        <p
          role="status"
          className="flex items-center gap-2 rounded-2xl bg-movo-ok/10 p-4 text-sm font-semibold text-movo-ink"
        >
          <CheckCircleIcon className="size-5 shrink-0 text-movo-ok" />
          Guardamos los cambios. Ya se ven en tu ficha.
        </p>
      ) : null}

      <button
        type="submit"
        disabled={form.pending}
        className="flex h-14 w-full items-center justify-center rounded-full bg-movo-accent font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-60"
      >
        {form.pending ? "Guardando…" : "Guardar cambios"}
      </button>
    </form>
  );
}
