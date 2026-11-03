"use client";

import { useEffect, useState } from "react";

import { Notice } from "@/components/onboarding/fields";
import { SearchIcon } from "@/components/socios/icons";
import { pageCount, pageWindow } from "@/lib/admin/pagination";
import {
  describeDaysLeft,
  formatDate,
  initials,
  paymentState,
} from "@/lib/admin/subscriptions/format";
import { useSubscriptions } from "@/lib/admin/subscriptions/hooks";
import {
  hasActiveFilters,
  NO_FILTERS,
  PAGE_SIZE,
  pageRange,
  STATUS_FILTERS,
} from "@/lib/admin/subscriptions/query";
import type { SubscriptionFilters } from "@/lib/admin/subscriptions/types";
import { useRequireSuperAdmin } from "@/lib/session/hooks";

import { AdminShell } from "./AdminShell";
import { SubscriptionDetail } from "./SubscriptionDetail";
import { PaymentStatePill, PlanChip, PrioritySupportMark } from "./SubscriptionPills";

const SEARCH_DELAY_MS = 300;

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
  const [filters, setFilters] = useState<SubscriptionFilters>(NO_FILTERS);
  const [draft, setDraft] = useState("");
  const [selected, setSelected] = useState<number | null>(null);
  const list = useSubscriptions(filters);

  // La búsqueda espera a que el admin deje de escribir para no pedir en cada tecla.
  useEffect(() => {
    const timer = setTimeout(() => {
      setFilters((current) =>
        current.query === draft ? current : { ...current, query: draft, page: 1 },
      );
    }, SEARCH_DELAY_MS);
    return () => clearTimeout(timer);
  }, [draft]);

  const data = list.state.status === "ready" ? list.state.data : null;
  const rows = data?.subscriptions ?? [];
  const total = data?.total ?? 0;
  const pages = pageCount(total, PAGE_SIZE);
  const { from, to } = pageRange(filters.page, total);
  const selectedRow = rows.find((row) => row.business_id === selected) ?? null;

  function change(next: Partial<SubscriptionFilters>) {
    setFilters((current) => ({ ...current, page: 1, ...next }));
  }

  function clear() {
    setDraft("");
    setFilters(NO_FILTERS);
  }

  return (
    <div className="space-y-6">
      <header>
        <h1 className="font-display text-3xl font-extrabold">Gestión de suscripciones y cobros</h1>
        <p className="mt-2 max-w-2xl text-movo-slate">
          Controlá las vigencias, registrá los pagos mensuales y avisá a los locales que están por
          vencer.
        </p>
      </header>

      <section className="space-y-4 rounded-3xl bg-white p-5 shadow-sm">
        <div className="flex items-center gap-3 rounded-2xl bg-movo-sand px-4 focus-within:ring-2 focus-within:ring-movo-accent">
          <SearchIcon className="size-5 shrink-0 text-movo-slate" />
          <input
            type="search"
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            maxLength={100}
            placeholder="Buscar por nombre del local…"
            aria-label="Buscar suscripciones"
            className="h-14 min-w-0 flex-1 bg-transparent text-base outline-none placeholder:text-movo-slate/60"
          />
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <div
            role="tablist"
            aria-label="Estado de la suscripción"
            className="flex flex-wrap gap-2"
          >
            {STATUS_FILTERS.map(({ value, label }) => (
              <button
                key={label}
                type="button"
                role="tab"
                aria-selected={value === filters.status}
                onClick={() => change({ status: value })}
                className={`h-10 rounded-full px-4 text-sm font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent ${
                  value === filters.status ? "bg-movo-card text-movo-accent" : "bg-movo-sand"
                }`}
              >
                {label}
              </button>
            ))}
          </div>

          <label className="flex items-center gap-2 text-sm font-semibold">
            Plan
            <select
              value={filters.plan}
              onChange={(event) =>
                change({ plan: event.target.value as SubscriptionFilters["plan"] })
              }
              className="h-10 rounded-full bg-movo-sand px-4 outline-none focus:ring-2 focus:ring-movo-accent"
            >
              <option value="">Todos</option>
              <option value="starter">Starter</option>
              <option value="pro">Pro</option>
            </select>
          </label>

          {hasActiveFilters(filters) || draft !== "" ? (
            <button
              type="button"
              onClick={clear}
              className="h-10 rounded-full px-4 text-sm font-bold underline outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
            >
              Limpiar filtros
            </button>
          ) : null}
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
              {rows.map((row) => (
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
                      <span>
                        <span className="block font-display text-base font-bold">
                          {row.business_name}
                        </span>
                        {row.priority_support ? <PrioritySupportMark /> : null}
                      </span>
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
          {data && rows.length === 0 ? (
            <p className="p-8 text-center text-movo-slate">
              {hasActiveFilters(filters)
                ? "No hay suscripciones que coincidan con los filtros."
                : "Todavía no hay suscripciones: aparecen cuando confirmás el primer pago de un local."}
            </p>
          ) : null}

          {total > 0 ? (
            <nav
              aria-label="Páginas"
              className="flex flex-wrap items-center justify-between gap-3 border-t border-movo-sand px-5 py-4"
            >
              <p className="text-sm text-movo-slate">
                Mostrando {from}–{to} de {total} suscripciones
              </p>
              <div className="flex items-center gap-1">
                <PageButton
                  disabled={filters.page === 1}
                  onClick={() => setFilters({ ...filters, page: filters.page - 1 })}
                  label="Anterior"
                >
                  ‹
                </PageButton>
                {pageWindow(filters.page, pages).map((n) => (
                  <PageButton
                    key={n}
                    current={n === filters.page}
                    onClick={() => setFilters({ ...filters, page: n })}
                    label={`Página ${n}`}
                  >
                    {n}
                  </PageButton>
                ))}
                <PageButton
                  disabled={filters.page >= pages}
                  onClick={() => setFilters({ ...filters, page: filters.page + 1 })}
                  label="Siguiente"
                >
                  ›
                </PageButton>
              </div>
            </nav>
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

function PageButton({
  children,
  label,
  current,
  disabled,
  onClick,
}: {
  children: string | number;
  label: string;
  current?: boolean;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      aria-current={current ? "page" : undefined}
      disabled={disabled}
      onClick={onClick}
      className={`grid size-10 place-items-center rounded-full text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-40 ${
        current ? "bg-movo-card text-movo-accent" : "bg-movo-sand"
      }`}
    >
      {children}
    </button>
  );
}
