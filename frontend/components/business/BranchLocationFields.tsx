"use client";

import dynamic from "next/dynamic";

import { Field, Notice, TextInput } from "@/components/onboarding/fields";
import { LinkIcon, MapPinIcon, NavigationIcon, XIcon } from "@/components/socios/icons";
import { MAX_REFERENCE_LENGTH } from "@/lib/business/types";
import type { useBranchForm } from "@/lib/business/useBranchForm";

// Leaflet necesita `window`, así que el mapa solo existe en el navegador.
const BranchMap = dynamic(() => import("@/components/onboarding/BranchMap"), {
  ssr: false,
  loading: () => <div className="size-full animate-pulse bg-movo-sand" aria-hidden="true" />,
});

interface Props {
  form: ReturnType<typeof useBranchForm>;
  /** Pregunta el nombre de la sede. El alta del negocio no lo hace: la primera se llama "Sede principal". */
  askName?: boolean;
}

/** Dirección, referencia, mapa con el pin y link de reservas: lo que define dónde está una sede. */
export function BranchLocationFields({ form, askName = false }: Props) {
  const { fields, form: notice } = form.feedback;
  const busy = form.pending || form.searching;

  return (
    <>
      <section className="space-y-4 rounded-3xl bg-white p-5 shadow-sm">
        {askName ? (
          <Field
            id="branch-name"
            label="Nombre de la sede"
            required
            error={fields.name}
            hint={
              <span className="text-xs font-semibold text-movo-slate">Visible en la búsqueda</span>
            }
          >
            <TextInput
              id="branch-name"
              value={form.name}
              onChange={form.setName}
              error={fields.name}
              disabled={busy}
              maxLength={100}
              placeholder="Sede Belgrano"
            />
          </Field>
        ) : null}

        <Field
          id="address"
          label="Dirección del local"
          required={askName}
          error={fields.address}
          hint={
            askName ? undefined : (
              <span className="text-xs font-bold text-movo-danger">Obligatorio</span>
            )
          }
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

        <Field
          id="reference"
          label="Piso / Depto / Timbre"
          error={fields.reference}
          hint={<span className="text-xs font-semibold text-movo-slate">Opcional</span>}
        >
          <TextInput
            id="reference"
            value={form.reference}
            onChange={form.setReference}
            error={fields.reference}
            disabled={busy}
            maxLength={MAX_REFERENCE_LENGTH}
            placeholder="Ej. PB - al fondo / Timbre 4B"
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
        <div
          className="relative isolate h-72"
          role="application"
          aria-label="Mapa para ubicar tu sede"
        >
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
    </>
  );
}
