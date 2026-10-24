"use client";

import dynamic from "next/dynamic";
import type { SubmitEvent } from "react";

import { LinkIcon, MapPinIcon, NavigationIcon, XIcon } from "@/components/socios/icons";
import type { Branch } from "@/lib/business/types";
import { useBranchForm } from "@/lib/business/useBranchForm";

import { Field, Notice, TextInput } from "./fields";
import { Shell, StepFooter } from "./Shell";

// Leaflet necesita `window`, así que el mapa solo existe en el navegador.
const BranchMap = dynamic(() => import("./BranchMap"), {
  ssr: false,
  loading: () => <div className="size-full animate-pulse bg-movo-sand" aria-hidden="true" />,
});

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
  const { fields, form: notice } = form.feedback;
  const busy = form.pending || form.searching;

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

      <form id="step-form" onSubmit={submit} noValidate aria-busy={busy} className="space-y-5">
        <section className="space-y-4 rounded-3xl bg-white p-5 shadow-sm">
          <Field
            id="address"
            label="Dirección del local"
            error={fields.address}
            hint={<span className="text-xs font-bold text-movo-danger">Obligatorio</span>}
          >
            <div className="relative">
              <TextInput
                id="address"
                value={form.address}
                onChange={form.setAddress}
                error={fields.address}
                disabled={busy}
                autoComplete="street-address"
                placeholder="Honduras 4820"
                maxLength={200}
              />
              {form.address ? (
                <button
                  type="button"
                  aria-label="Borrar la dirección"
                  onClick={() => form.setAddress("")}
                  className="absolute top-1/2 right-3 grid size-8 -translate-y-1/2 place-items-center rounded-full bg-white text-movo-slate"
                >
                  <XIcon className="size-4" />
                </button>
              ) : null}
            </div>
          </Field>

          <Field id="city" label="Ciudad o localidad" error={fields.city}>
            <TextInput
              id="city"
              value={form.city}
              onChange={form.setCity}
              error={fields.city}
              disabled={busy}
              autoComplete="address-level2"
              placeholder="Buenos Aires"
              maxLength={100}
            />
          </Field>

          <div className="flex flex-wrap gap-3">
            <button
              type="button"
              onClick={() => void form.search()}
              disabled={busy}
              className="flex h-12 items-center gap-2 rounded-full bg-movo-card px-5 font-semibold text-movo-accent outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-60"
            >
              <MapPinIcon className="size-4" />
              {form.searching ? "Buscando…" : "Buscar en el mapa"}
            </button>
            <button
              type="button"
              onClick={form.locateMe}
              disabled={busy}
              className="flex h-12 items-center gap-2 rounded-full bg-movo-sand px-5 font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-60"
            >
              <NavigationIcon className="size-4" />
              Usar mi ubicación actual
            </button>
          </div>
        </section>

        <section className="overflow-hidden rounded-3xl bg-white shadow-sm">
          <div className="relative h-72" role="application" aria-label="Mapa para ubicar tu sede">
            <BranchMap pin={form.pin} view={form.view} onMove={form.movePin} />
            <p className="pointer-events-none absolute top-3 left-1/2 z-[1000] -translate-x-1/2 rounded-full bg-movo-card/90 px-4 py-2 text-xs font-semibold whitespace-nowrap text-white">
              Arrastrá el pin o tocá el mapa para ajustar la entrada
            </p>
          </div>
          <div className="flex items-center justify-between gap-3 px-5 py-3 text-sm">
            <p className={fields.location ? "font-semibold text-movo-danger" : "text-movo-slate"}>
              {fields.location ??
                (form.pin
                  ? `Lat ${form.pin.latitude.toFixed(4)} · Lng ${form.pin.longitude.toFixed(4)}`
                  : "Buscá la dirección o tocá el mapa para poner el pin")}
            </p>
            <button
              type="button"
              onClick={form.recenter}
              disabled={!form.pin}
              className="shrink-0 font-bold text-movo-gold underline underline-offset-4 disabled:opacity-40"
            >
              Recentrar
            </button>
          </div>
        </section>

        <section className="rounded-3xl bg-white p-5 shadow-sm">
          <Field
            id="booking"
            label="Link de reservas online"
            error={fields.booking_url}
            hint={<span className="text-xs font-semibold text-movo-slate">Opcional</span>}
            help='Para el botón directo "Reservar" de tu ficha (ej. TurnoWeb, EasyCancha o tu web propia).'
          >
            <div className="flex items-center gap-3 rounded-2xl bg-movo-sand pl-4 focus-within:ring-2 focus-within:ring-movo-accent">
              <LinkIcon className="size-5 shrink-0 text-movo-slate" />
              <input
                id="booking"
                name="booking"
                type="url"
                inputMode="url"
                value={form.bookingUrl}
                onChange={(event) => form.setBookingUrl(event.target.value)}
                disabled={busy}
                placeholder="https://turnos.tuclub.com"
                aria-invalid={fields.booking_url ? true : undefined}
                aria-describedby={fields.booking_url ? "booking-error" : "booking-help"}
                className="h-14 min-w-0 flex-1 bg-transparent pr-4 text-base outline-none placeholder:text-movo-slate/60"
              />
            </div>
          </Field>
        </section>

        {notice ? <Notice title={notice.title} body={notice.body} /> : null}
      </form>

      <StepFooter onBack={onBack} pending={form.pending} disabled={form.searching} />
    </Shell>
  );
}
