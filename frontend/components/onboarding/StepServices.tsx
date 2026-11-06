"use client";

import { useState, type SubmitEvent } from "react";

import { ServiceSheet } from "@/components/business/ServiceSheet";
import { ClockIcon, PencilIcon, PlusIcon, TrashIcon } from "@/components/socios/icons";
import { removeService } from "@/lib/business/api";
import { useServices } from "@/lib/business/hooks";
import { describeFailure, type FormMessage } from "@/lib/business/messages";
import { describeSchedule } from "@/lib/business/schedule";
import type { Service } from "@/lib/business/types";

import { Notice } from "./fields";
import { Shell, StepFooter } from "./Shell";

interface Props {
  onNext: () => void;
  onBack: () => void;
}

/** Lo que está abierto en la hoja inferior: nada, un servicio nuevo o uno que se edita. */
type Editing = { open: false } | { open: true; service: Service | null };

/** Paso 3: los servicios del negocio con sus días y franjas. */
export function StepServices({ onNext, onBack }: Props) {
  const services = useServices();
  const [editing, setEditing] = useState<Editing>({ open: false });
  const [message, setMessage] = useState<FormMessage | null>(null);

  function next(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    onNext();
  }

  async function remove(service: Service) {
    const result = await removeService(service.id);
    setMessage(result.ok ? null : (describeFailure(result).form ?? null));
    services.reload();
  }

  const list = services.state.status === "ready" ? services.state.data : [];

  return (
    <Shell step={3}>
      <section>
        <h2 className="font-display text-4xl font-extrabold">Tus servicios</h2>
        <p className="mt-2 text-base text-movo-slate">
          Cargá lo que ofrecés y en qué días y horarios. Podés sumar más después desde tu panel.
        </p>
      </section>

      <form id="step-form" onSubmit={next} className="space-y-4">
        {services.state.status === "failed" ? (
          <Notice
            title="No pudimos cargar tus servicios"
            body="Recargá la página para volver a intentar."
          />
        ) : null}
        {message ? <Notice title={message.title} body={message.body} /> : null}

        {services.state.status === "ready" && list.length === 0 ? (
          <p className="rounded-3xl bg-white p-6 text-center text-movo-slate shadow-sm">
            Todavía no cargaste ningún servicio.
          </p>
        ) : null}

        <ul className="space-y-3">
          {list.map((service) => (
            <li
              key={service.id}
              className="flex items-center gap-3 rounded-3xl bg-white p-5 shadow-sm"
            >
              <div className="min-w-0 flex-1">
                <p className="truncate font-display text-lg font-bold">{service.name}</p>
                <p className="mt-1 flex items-center gap-2 text-sm text-movo-slate">
                  <ClockIcon className="size-4 shrink-0" />
                  {describeSchedule(service.schedule) || "Sin horarios cargados"}
                </p>
              </div>
              <button
                type="button"
                aria-label={`Editar ${service.name}`}
                onClick={() => setEditing({ open: true, service })}
                className="grid size-10 place-items-center rounded-full bg-movo-sand outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
              >
                <PencilIcon className="size-4" />
              </button>
              <button
                type="button"
                aria-label={`Quitar ${service.name}`}
                onClick={() => void remove(service)}
                className="grid size-10 place-items-center rounded-full bg-movo-sand text-movo-danger outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
              >
                <TrashIcon className="size-4" />
              </button>
            </li>
          ))}
        </ul>

        <button
          type="button"
          onClick={() => setEditing({ open: true, service: null })}
          className="flex h-14 w-full items-center justify-center gap-2 rounded-full border-2 border-dashed border-movo-slate/40 font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
        >
          <PlusIcon />
          Agregar servicio
        </button>
      </form>

      {editing.open ? (
        <ServiceSheet
          // La clave hace que cada servicio abra con su propio formulario limpio.
          key={editing.service?.id ?? "new"}
          service={editing.service}
          onClose={() => setEditing({ open: false })}
          onSaved={services.reload}
        />
      ) : null}

      <StepFooter onBack={onBack} />
    </Shell>
  );
}
