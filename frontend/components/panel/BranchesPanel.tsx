"use client";

import Link from "next/link";
import { useState } from "react";

import { Notice } from "@/components/onboarding/fields";
import { MapPinIcon, PencilIcon, PlusIcon } from "@/components/socios/icons";
import { setBranchActive } from "@/lib/business/api";
import { useBusiness } from "@/lib/business/BusinessProvider";
import { useServices } from "@/lib/business/hooks";
import { describeFailure, type FormMessage } from "@/lib/business/messages";
import type { Branch } from "@/lib/business/types";

import { PlanUsage } from "./PlanUsage";
import { Switch } from "./Switch";

/** Mis sedes: el uso del plan, la lista con su interruptor de visibilidad y el alta. */
export function BranchesPanel() {
  const { business, plan, reload } = useBusiness();
  const services = useServices();
  const [message, setMessage] = useState<FormMessage | null>(null);
  const [busyId, setBusyId] = useState<number | null>(null);

  const branches = business.status === "ready" ? business.data.branches : [];
  const activeCount = branches.filter((branch) => branch.active).length;
  const limit = plan.status === "ready" ? plan.data.max_branches : null;
  const full = limit !== null && activeCount >= limit;
  const serviceList = services.state.status === "ready" ? services.state.data : [];

  async function toggle(branch: Branch, active: boolean) {
    setBusyId(branch.id);
    setMessage(null);

    const result = await setBranchActive(branch.id, active);
    if (!result.ok) setMessage(describeFailure(result).form ?? null);

    setBusyId(null);
    reload();
  }

  return (
    <div className="space-y-5">
      <p className="text-base text-movo-slate">
        Gestioná las ubicaciones físicas de tu negocio donde los clientes pueden encontrarte.
      </p>

      {plan.status === "ready" ? (
        <PlanUsage plan={plan.data} used={activeCount} limit={limit} noun="sedes" inUse="activas" />
      ) : null}

      {full ? (
        <p className="flex h-14 items-center justify-center rounded-full bg-movo-accent/50 px-4 text-center text-sm font-bold">
          Llegaste al límite de sedes de tu plan
        </p>
      ) : (
        <Link
          href="/socios/sedes/nueva"
          className="flex h-14 items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
        >
          <PlusIcon />
          Agregar sede
        </Link>
      )}

      {message ? <Notice title={message.title} body={message.body} /> : null}

      <div className="flex items-baseline justify-between">
        <h2 className="font-display text-lg font-bold">Tus sedes configuradas</h2>
        <p className="text-sm text-movo-slate">
          {activeCount} {activeCount === 1 ? "activa" : "activas"}
        </p>
      </div>

      {business.status === "ready" && branches.length === 0 ? (
        <p className="rounded-3xl bg-white p-6 text-center text-movo-slate shadow-sm">
          Todavía no cargaste ninguna sede.
        </p>
      ) : null}

      <ul className="space-y-3">
        {branches.map((branch) => {
          const count = serviceList.filter((service) => service.branch_id === branch.id).length;
          return (
            <li
              key={branch.id}
              className={`rounded-3xl bg-white p-5 shadow-sm ${branch.active ? "" : "opacity-70"}`}
            >
              <div className="flex items-start gap-4">
                <span
                  className="grid size-16 shrink-0 place-items-center rounded-2xl bg-movo-sand text-movo-gold"
                  aria-hidden="true"
                >
                  <MapPinIcon className="size-7" />
                </span>
                <div className="min-w-0 flex-1">
                  <div className="flex items-start justify-between gap-2">
                    <h3 className="font-display text-xl leading-tight font-bold">{branch.name}</h3>
                    <span
                      className={`shrink-0 rounded-full px-3 py-1 text-xs font-bold ${
                        branch.active
                          ? "bg-movo-ok/15 text-movo-ok"
                          : "bg-movo-sand text-movo-slate"
                      }`}
                    >
                      {branch.active ? "Activa" : "Oculta"}
                    </span>
                  </div>
                  <p className="mt-1 text-sm text-movo-slate">
                    {branch.address}, {branch.city}
                  </p>
                  <ul className="mt-2 flex flex-wrap gap-2 text-xs font-semibold">
                    {branch.reference ? (
                      <li className="rounded-full bg-movo-sand px-3 py-1">{branch.reference}</li>
                    ) : null}
                    <li className="rounded-full bg-movo-sand px-3 py-1">
                      {count} {count === 1 ? "servicio" : "servicios"}
                    </li>
                  </ul>
                </div>
              </div>

              <div className="mt-4 flex items-center justify-between gap-3 border-t border-movo-sand pt-4">
                <label className="flex items-center gap-3 text-sm font-semibold">
                  <Switch
                    checked={branch.active}
                    label={`${branch.name}: ${branch.active ? "visible al público" : "oculta"}`}
                    disabled={busyId === branch.id}
                    onChange={(next) => void toggle(branch, next)}
                  />
                  Visible al público
                </label>
                <Link
                  href={`/socios/sedes/${branch.id}`}
                  aria-label={`Editar ${branch.name}`}
                  className="grid size-10 place-items-center rounded-full bg-movo-sand outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
                >
                  <PencilIcon className="size-4" />
                </Link>
              </div>
            </li>
          );
        })}
      </ul>

      <p className="rounded-3xl bg-movo-sand p-4 text-sm text-movo-slate">
        Los cambios en direcciones y horarios se ven en el mapa apenas los guardás.
      </p>
    </div>
  );
}
