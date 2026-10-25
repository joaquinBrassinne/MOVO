"use client";

import { useEffect, useRef, type SubmitEvent } from "react";

import { Field, Notice, TextArea, TextInput } from "@/components/onboarding/fields";
import { CheckIcon, PlusIcon, XIcon } from "@/components/socios/icons";
import { WEEK } from "@/lib/business/schedule";
import type { Service } from "@/lib/business/types";
import { useServiceForm } from "@/lib/business/useServiceForm";

interface Props {
  /** El servicio a editar, o `null` para crear uno nuevo. */
  service: Service | null;
  onClose: () => void;
  /** Se llama después de guardar, antes de cerrar. */
  onSaved: () => void;
  /** Si está, la hoja ofrece borrar el servicio (solo al editar). */
  onRemove?: () => Promise<void>;
}

/** Hoja inferior para crear o editar un servicio con sus días y franjas. */
export function ServiceSheet({ service, onClose, onSaved, onRemove }: Props) {
  const form = useServiceForm(service);
  const dialog = useRef<HTMLDialogElement>(null);
  const { fields, form: notice } = form.feedback;

  // <dialog> da el foco atrapado, el cierre con Escape y el fondo oscuro sin código extra.
  useEffect(() => {
    const element = dialog.current;
    if (element && !element.open) element.showModal();
  }, []);

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    if (await form.submit()) {
      onSaved();
      onClose();
    }
  }

  return (
    <dialog
      ref={dialog}
      onClose={onClose}
      aria-labelledby="service-sheet-title"
      className="fixed inset-x-0 bottom-0 mx-auto mt-auto max-h-[92dvh] w-full max-w-md overflow-y-auto rounded-t-[2rem] bg-white p-0 text-movo-ink backdrop:bg-black/50"
    >
      <form onSubmit={submit} noValidate aria-busy={form.pending} className="space-y-5 p-6 pb-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-xs font-bold tracking-wide text-movo-gold uppercase">
              {service ? "Editar ítem" : "Nuevo ítem"}
            </p>
            <h2 id="service-sheet-title" className="font-display text-2xl font-extrabold">
              {service ? "Editar servicio" : "Nuevo servicio"}
            </h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Cerrar"
            className="grid size-10 place-items-center rounded-full bg-movo-sand outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
          >
            <XIcon />
          </button>
        </div>

        <Field id="service-name" label="Nombre del servicio" error={fields.name}>
          <TextInput
            id="service-name"
            value={form.name}
            onChange={form.setName}
            error={fields.name}
            disabled={form.pending}
            maxLength={100}
            placeholder="Yoga Vinyasa Flow"
          />
        </Field>

        <Field id="service-description" label="Descripción corta" error={fields.description}>
          <TextArea
            id="service-description"
            value={form.description}
            onChange={form.setDescription}
            error={fields.description}
            disabled={form.pending}
            maxLength={500}
            rows={3}
          />
        </Field>

        <Field
          id="service-price"
          label="Precio estimado"
          error={fields.price}
          hint={<span className="text-xs font-semibold text-movo-slate">Opcional · en pesos</span>}
        >
          <TextInput
            id="service-price"
            value={form.price}
            onChange={form.setPrice}
            error={fields.price}
            disabled={form.pending}
            inputMode="numeric"
            placeholder="5500"
            maxLength={12}
          />
        </Field>

        {form.scheduleEditable ? (
          <>
            <Field
              id="service-days"
              label="Días disponibles"
              error={fields.days}
              hint={
                <span className="text-xs font-bold text-movo-gold">
                  {form.days.length} seleccionados
                </span>
              }
            >
              <div id="service-days" className="flex justify-between gap-1" role="group">
                {WEEK.map(({ weekday, short, long }) => {
                  const selected = form.days.includes(weekday);
                  return (
                    <button
                      key={weekday}
                      type="button"
                      aria-pressed={selected}
                      aria-label={long}
                      disabled={form.pending}
                      onClick={() => form.toggleDay(weekday)}
                      className={`grid size-11 place-items-center rounded-full text-xs font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent ${
                        selected ? "bg-movo-card text-movo-accent" : "bg-movo-sand"
                      }`}
                    >
                      {short}
                    </button>
                  );
                })}
              </div>
            </Field>

            <Field id="service-ranges" label="Franjas horarias" error={fields.slots}>
              <div id="service-ranges" className="flex flex-wrap items-center gap-2">
                {form.ranges.map((range, index) => (
                  <span
                    key={`${range.start}-${range.end}`}
                    className="inline-flex h-10 items-center gap-2 rounded-full bg-movo-card pr-2 pl-4 text-sm font-bold text-white"
                  >
                    {range.start} a {range.end}
                    <button
                      type="button"
                      aria-label={`Quitar la franja de ${range.start} a ${range.end}`}
                      onClick={() => form.removeRange(index)}
                      className="grid size-6 place-items-center rounded-full text-movo-accent outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
                    >
                      <XIcon className="size-4" />
                    </button>
                  </span>
                ))}

                {form.drafting ? null : (
                  <button
                    type="button"
                    onClick={form.startDraft}
                    className="inline-flex h-10 items-center gap-2 rounded-full bg-movo-sand px-4 text-sm font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
                  >
                    <PlusIcon className="size-4" />
                    Agregar franja
                  </button>
                )}
              </div>

              {form.drafting ? (
                <div className="mt-3 flex items-end gap-2">
                  <label className="flex-1 text-xs font-semibold text-movo-slate">
                    Desde
                    <input
                      type="time"
                      value={form.draft.start}
                      onChange={(event) =>
                        form.setDraft({ ...form.draft, start: event.target.value })
                      }
                      className="mt-1 h-12 w-full rounded-2xl bg-movo-sand px-3 text-base text-movo-ink outline-none focus:ring-2 focus:ring-movo-accent"
                    />
                  </label>
                  <label className="flex-1 text-xs font-semibold text-movo-slate">
                    Hasta
                    <input
                      type="time"
                      value={form.draft.end}
                      onChange={(event) =>
                        form.setDraft({ ...form.draft, end: event.target.value })
                      }
                      className="mt-1 h-12 w-full rounded-2xl bg-movo-sand px-3 text-base text-movo-ink outline-none focus:ring-2 focus:ring-movo-accent"
                    />
                  </label>
                  <button
                    type="button"
                    onClick={form.commitDraft}
                    className="h-12 rounded-full bg-movo-card px-5 text-sm font-bold text-movo-accent outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
                  >
                    Sumar
                  </button>
                </div>
              ) : null}
            </Field>
          </>
        ) : (
          <p className="rounded-2xl bg-movo-sand p-4 text-sm text-movo-slate">
            Este servicio tiene horarios distintos según el día, así que por ahora no se pueden
            editar desde acá. Los demás datos sí.
          </p>
        )}

        {notice ? <Notice title={notice.title} body={notice.body} /> : null}

        <button
          type="submit"
          disabled={form.pending}
          className="flex h-14 w-full items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-60"
        >
          {form.pending ? "Guardando…" : "Guardar"}
          {form.pending ? null : <CheckIcon />}
        </button>

        {onRemove ? (
          <button
            type="button"
            disabled={form.pending}
            onClick={async () => {
              if (!window.confirm("¿Eliminar este servicio? No se puede deshacer.")) return;
              await onRemove();
              onClose();
            }}
            className="h-12 w-full text-sm font-bold text-movo-danger underline underline-offset-4 disabled:opacity-60"
          >
            Eliminar servicio
          </button>
        ) : null}
      </form>
    </dialog>
  );
}
