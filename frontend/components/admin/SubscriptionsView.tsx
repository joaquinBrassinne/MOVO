"use client";

import { useState } from "react";

import { Notice } from "@/components/onboarding/fields";
import { SearchIcon } from "@/components/socios/icons";
import {
  describeDaysLeft,
  filterRows,
  formatDate,
  initials,
  paymentState,
  TAB_LABEL,
  TAB_ORDER,
  tabCounts,
} from "@/lib/admin/subscriptions/format";
import { useSubscriptions } from "@/lib/admin/subscriptions/hooks";
import type { PaymentState, SubscriptionTab } from "@/lib/admin/subscriptions/types";
import { useRequireSuperAdmin } from "@/lib/session/hooks";

import { AdminShell } from "./AdminShell";
import { SubscriptionDetail } from "./SubscriptionDetail";
import { PaymentStatePill, PlanChip } from "./SubscriptionPills";

const SUMMARY: { state: PaymentState; label: string; hint: string }[] = [
  { state: "ok", label: "Al día", hint: "con el plan vigente" },
  { state: "expiring", label: "Por vencer", hint: "vencen en los próximos 7 días" },
  { state: "grace", label: "En gracia", hint: "ya vencieron, con plazo adicional" },
];

/** Gestión de suscripciones y cobros: vigencias, estados y acciones de cobro del super admin. */
export function SubscriptionsView() {
  const allowed = useRequireSuperAdmin();
  if (!allowed) return null;

  return (
    <AdminShell>
      <Subscriptions />
    </AdminShell>
  );
}

function Subscriptions() {
  const list = useSubscriptions();
  const [tab, setTab] = useState<SubscriptionTab>("all");
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<number | null>(null);

  const rows = list.state.status === "ready" ? list.state.data : [];
  const counts = tabCounts(rows);
  const visible = filterRows(rows, tab, search);
  const selectedRow = rows.find((row) => row.business_id === selected) ?? null;

  return (
    <div className="space-y-6">
      <header>
        <h1 className="font-display text-3xl font-extrabold">Gestión de suscripciones y cobros</h1>
        <p className="mt-2 max-w-2xl text-movo-slate">
          Controlá las vigencias, registrá los pagos mensuales y avisá a los locales que están por
          vencer.
        </p>
      </header>

      <ul className="grid gap-4 sm:grid-cols-3" aria-label="Resumen">
        {SUMMARY.map(({ state, label, hint }) => (
          <li key={state} className="rounded-3xl bg-white p-5 shadow-sm">
            <p className="text-xs font-bold tracking-wide text-movo-slate uppercase">{label}</p>
            <p className="mt-2 font-display text-4xl font-extrabold">{counts[state]}</p>
            <p className="text-sm text-movo-slate">{hint}</p>
          </li>
        ))}
      </ul>

      <section className="space-y-4 rounded-3xl bg-white p-5 shadow-sm">
        <div className="flex items-center gap-3 rounded-2xl bg-movo-sand px-4 focus-within:ring-2 focus-within:ring-movo-accent">
          <SearchIcon className="size-5 shrink-0 text-movo-slate" />
          <input
            type="search"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            maxLength={100}
            placeholder="Buscar por nombre del local…"
            aria-label="Buscar suscripciones"
            className="h-14 min-w-0 flex-1 bg-transparent text-base outline-none placeholder:text-movo-slate/60"
          />
        </div>

        <div role="tablist" aria-label="Estado de la suscripción" className="flex flex-wrap gap-2">
          {TAB_ORDER.map((key) => (
            <button
              key={key}
              type="button"
              role="tab"
              aria-selected={key === tab}
              onClick={() => setTab(key)}
              className={`h-10 rounded-full px-4 text-sm font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent ${
                key === tab ? "bg-movo-card text-movo-accent" : "bg-movo-sand"
              }`}
            >
              {TAB_LABEL[key]} ({counts[key]})
            </button>
          ))}
        </div>
      </section>

      <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_26rem]">
        <section className="overflow-hidden rounded-3xl bg-white shadow-sm">
          {list.state.status === "failed" ? (
            <div className="p-5">
              <Notice
                title="No pudimos cargar las suscripciones"
                body="Recargá la página para volver a intentar."
              />
            </div>
          ) : null}

          <table className="w-full text-left text-sm">
            <thead className="bg-movo-sand text-xs font-bold tracking-wide text-movo-slate uppercase">
              <tr>
                <th scope="col" className="px-5 py-3">
                  Local
                </th>
                <th scope="col" className="px-5 py-3">
                  Plan
                </th>
                <th scope="col" className="px-5 py-3">
                  Vigencia
                </th>
                <th scope="col" className="px-5 py-3">
                  Estado
                </th>
                <th scope="col" className="px-5 py-3">
                  <span className="sr-only">Acciones</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {visible.map((row) => (
                <tr
                  key={row.business_id}
                  className={`border-t border-movo-sand ${row.business_id === selected ? "bg-movo-accent/15" : ""}`}
                >
                  <th scope="row" className="px-5 py-4 text-left font-normal">
                    <span className="flex items-center gap-3">
                      <span
                        aria-hidden="true"
                        className="grid size-10 shrink-0 place-items-center rounded-full bg-movo-card text-sm font-bold text-movo-accent"
                      >
                        {initials(row.business_name)}
                      </span>
                      <span className="font-display text-base font-bold">{row.business_name}</span>
                    </span>
                  </th>
                  <td className="px-5 py-4">
                    <PlanChip plan={row.plan} />
                  </td>
                  <td className="px-5 py-4">
                    <span className="block font-semibold">{formatDate(row.paid_until)}</span>
                    <span className="block text-xs text-movo-slate">
                      {describeDaysLeft(row.days_left)}
                    </span>
                  </td>
                  <td className="px-5 py-4">
                    <PaymentStatePill state={paymentState(row)} />
                  </td>
                  <td className="px-5 py-4 text-right">
                    <button
                      type="button"
                      aria-label={`Ver detalle de ${row.business_name}`}
                      aria-pressed={row.business_id === selected}
                      onClick={() => setSelected(row.business_id)}
                      className="h-10 rounded-full bg-movo-sand px-4 text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
                    >
                      Gestionar
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          {list.state.status === "loading" ? (
            <p className="p-8 text-center text-movo-slate">Cargando…</p>
          ) : null}
          {list.state.status === "ready" && visible.length === 0 ? (
            <p className="p-8 text-center text-movo-slate">
              {rows.length === 0
                ? "Todavía no hay suscripciones: aparecen cuando confirmás el primer pago de un local."
                : "No hay suscripciones que coincidan con los filtros."}
            </p>
          ) : null}
          {list.state.status === "ready" && rows.length > 0 ? (
            <p className="border-t border-movo-sand px-5 py-4 text-sm text-movo-slate">
              Mostrando {visible.length} de {rows.length} suscripciones
            </p>
          ) : null}
        </section>

        {selectedRow ? (
          <SubscriptionDetail
            key={selectedRow.business_id}
            row={selectedRow}
            onClose={() => setSelected(null)}
            onChanged={list.reload}
          />
        ) : null}
      </div>
    </div>
  );
}
