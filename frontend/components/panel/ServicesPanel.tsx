"use client";

import { useState } from "react";

import { ServiceSheet } from "@/components/business/ServiceSheet";
import { Notice } from "@/components/onboarding/fields";
import { ClockIcon, MapPinIcon, PencilIcon, PlusIcon } from "@/components/socios/icons";
import { removeService, setServiceActive } from "@/lib/business/api";
import { useBusiness } from "@/lib/business/BusinessProvider";
import { useServices } from "@/lib/business/hooks";
import { describeFailure, type FormMessage } from "@/lib/business/messages";
import { formatPrice } from "@/lib/business/price";
import { describeSchedule } from "@/lib/business/schedule";
import type { Service } from "@/lib/business/types";

import { PlanUsage } from "./PlanUsage";
import { Switch } from "./Switch";

/** Lo que está abierto en la hoja inferior: nada, un servicio nuevo o uno que se edita. */
type Editing = { open: false } | { open: true; service: Service | null };

/** Servicios y horarios: el uso del plan, la lista con su interruptor y el alta. */
export function ServicesPanel() {
  const { business, plan } = useBusiness();
  const services = useServices();
  const [editing, setEditing] = useState<Editing>({ open: false });
  const [message, setMessage] = useState<FormMessage | null>(null);
  const [busyId, setBusyId] = useState<number | null>(null);

  const list = services.state.status === "ready" ? services.state.data : [];
  const branches = business.status === "ready" ? business.data.branches : [];
  const activeCount = list.filter((service) => service.active).length;
  const limit = plan.status === "ready" ? plan.data.max_active_services : null;
  const full = limit !== null && activeCount >= limit;

  async function toggle(service: Service, active: boolean) {
    setBusyId(service.id);
    setMessage(null);

    const result = await setServiceActive(service.id, active);
    if (!result.ok) setMessage(describeFailure(result).form ?? null);

    setBusyId(null);
    services.reload();
  }

  async function remove(service: Service) {
    const result = await removeService(service.id);
    if (!result.ok) setMessage(describeFailure(result).form ?? null);
    services.reload();
  }

  return (
    <div className="space-y-5">
      <p className="text-base text-movo-slate">Controlá la oferta que ven los clientes en MOVO.</p>

      {plan.status === "ready" ? (
        <PlanUsage
          plan={plan.data}
          used={activeCount}
          limit={limit}
          noun="servicios"
          inUse="activos"
        />
      ) : null}

      <button
        type="button"
        onClick={() => setEditing({ open: true, service: null })}
        disabled={full || services.state.status !== "ready"}
        className="flex h-14 w-full items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-50"
      >
        <PlusIcon />
        Agregar servicio
      </button>
      {full ? (
        <p className="-mt-2 text-center text-sm text-movo-slate">
          Pausá uno de tus servicios para poder agregar otro.
        </p>
      ) : null}

      {services.state.status === "failed" ? (
        <Notice
          title="No pudimos cargar tus servicios"
          body="Recargá la página para volver a intentar."
        />
      ) : null}
      {message ? <Notice title={message.title} body={message.body} /> : null}

      <div className="flex items-baseline justify-between">
        <h2 className="font-display text-lg font-bold">Tus servicios configurados</h2>
        <p className="text-sm text-movo-slate">
          {activeCount} {activeCount === 1 ? "activo" : "activos"} · {list.length - activeCount}{" "}
          {list.length - activeCount === 1 ? "pausado" : "pausados"}
        </p>
      </div>

      {services.state.status === "ready" && list.length === 0 ? (
        <p className="rounded-3xl bg-white p-6 text-center text-movo-slate shadow-sm">
          Todavía no cargaste ningún servicio.
        </p>
      ) : null}

      <ul className="space-y-3">
        {list.map((service) => {
          const branch = branches.find((b) => b.id === service.branch_id);
          const schedule = describeSchedule(service.schedule);
          return (
            <li
              key={service.id}
              className={`rounded-3xl bg-white p-5 shadow-sm ${service.active ? "" : "opacity-70"}`}
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <p className="text-xs font-bold tracking-wide text-movo-slate uppercase">
                    {service.active ? "Activo" : "Pausado temporalmente"}
                  </p>
                  <h3 className="mt-1 font-display text-xl leading-tight font-bold">
                    {service.name}
                  </h3>
                  <p className="mt-1 flex items-center gap-1 text-sm text-movo-slate">
                    <MapPinIcon className="size-4 shrink-0" />
                    <span className="truncate">{branch?.name ?? "Todas las sedes"}</span>
                  </p>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <button
                    type="button"
                    aria-label={`Editar ${service.name}`}
                    onClick={() => setEditing({ open: true, service })}
                    className="grid size-10 place-items-center rounded-full bg-movo-sand outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
                  >
                    <PencilIcon className="size-4" />
                  </button>
                  <Switch
                    checked={service.active}
                    label={`${service.name}: ${service.active ? "activo" : "pausado"}`}
                    disabled={busyId === service.id}
                    onChange={(next) => void toggle(service, next)}
                  />
                </div>
              </div>

              <div className="mt-4 flex items-center justify-between gap-3 rounded-2xl bg-movo-cream px-4 py-3 text-sm">
                <p className="flex min-w-0 items-center gap-2">
                  <ClockIcon className="size-4 shrink-0" />
                  <span className="truncate">{schedule || "Sin horarios cargados"}</span>
                </p>
                {service.price ? (
                  <p className="shrink-0 font-display font-bold">{formatPrice(service.price)}</p>
                ) : null}
              </div>
            </li>
          );
        })}
      </ul>

      <p className="flex items-start gap-2 rounded-3xl bg-movo-sand p-4 text-sm text-movo-slate">
        Los cambios se ven en tu ficha apenas los guardás.
      </p>

      {editing.open ? (
        <ServiceSheet
          // La clave hace que cada servicio abra con su propio formulario limpio.
          key={editing.service?.id ?? "new"}
          service={editing.service}
          onClose={() => setEditing({ open: false })}
          onSaved={services.reload}
          onRemove={editing.service ? () => remove(editing.service as Service) : undefined}
        />
      ) : null}
    </div>
  );
}
