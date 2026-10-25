"use client";

import { useEffect, useState } from "react";

import { Notice } from "@/components/onboarding/fields";
import { SearchIcon } from "@/components/socios/icons";
import { useAdminDirectory } from "@/lib/admin/hooks";
import { pageCount, pageWindow } from "@/lib/admin/pagination";
import { PAGE_SIZE, type DirectoryFilters } from "@/lib/admin/types";
import { useCategories } from "@/lib/business/hooks";
import { BUSINESS_STATUS, type BusinessStatus } from "@/lib/business/types";
import { useRequireSuperAdmin } from "@/lib/session/hooks";

import { AdminShell } from "./AdminShell";
import { BusinessDetail } from "./BusinessDetail";
import { StatusPill } from "./StatusPill";

const SEARCH_DELAY_MS = 300;

const TABS: { label: string; status: DirectoryFilters["status"] }[] = [
  { label: "Todos", status: "" },
  { label: "En revisión", status: BUSINESS_STATUS.PENDING_REVIEW },
  { label: "Activos", status: BUSINESS_STATUS.ACTIVE },
  { label: "Pausados", status: BUSINESS_STATUS.PAUSED },
];

/** Gestión de locales: directorio con búsqueda y filtros, y el detalle del elegido. */
export function LocalesView() {
  const allowed = useRequireSuperAdmin();
  if (!allowed) return null;

  return (
    <AdminShell>
      <Directory />
    </AdminShell>
  );
}

function Directory() {
  const [status, setStatus] = useState<DirectoryFilters["status"]>("");
  const [draft, setDraft] = useState("");
  const [query, setQuery] = useState("");
  const [categoryId, setCategoryId] = useState(0);
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState<number | null>(null);

  const directory = useAdminDirectory({ status, query, categoryId, page });
  const categories = useCategories();

  // La búsqueda espera a que el admin deje de escribir para no pedir en cada tecla.
  useEffect(() => {
    const timer = setTimeout(() => {
      setQuery(draft);
      setPage(1);
    }, SEARCH_DELAY_MS);
    return () => clearTimeout(timer);
  }, [draft]);

  const data = directory.state.status === "ready" ? directory.state.data : null;
  const counts = data?.counts ?? {};
  const registered = Object.entries(counts)
    .filter(([key]) => key !== BUSINESS_STATUS.DELETED)
    .reduce((sum, [, n]) => sum + (n ?? 0), 0);
  const pages = data ? pageCount(data.total, PAGE_SIZE) : 0;
  const from = data && data.total > 0 ? (page - 1) * PAGE_SIZE + 1 : 0;
  const to = data ? Math.min(page * PAGE_SIZE, data.total) : 0;

  function chooseStatus(next: DirectoryFilters["status"]) {
    setStatus(next);
    setPage(1);
  }

  return (
    <div className="space-y-6">
      <header>
        <h1 className="flex flex-wrap items-center gap-3 font-display text-3xl font-extrabold">
          Gestión de locales
          <span className="rounded-full bg-movo-card px-3 py-1 text-sm font-bold text-movo-accent">
            {registered} locales registrados
          </span>
        </h1>
        <p className="mt-2 max-w-2xl text-movo-slate">
          Revisá las solicitudes de alta, aprobá comercios y gestioná su estado.
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
            placeholder="Buscar por nombre, email del dueño, dirección o ID de local…"
            aria-label="Buscar locales"
            className="h-14 min-w-0 flex-1 bg-transparent text-base outline-none placeholder:text-movo-slate/60"
          />
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <div role="tablist" aria-label="Estado" className="flex flex-wrap gap-2">
            {TABS.map((tab) => {
              const count =
                tab.status === "" ? registered : (counts[tab.status as BusinessStatus] ?? 0);
              const active = tab.status === status;
              return (
                <button
                  key={tab.label}
                  type="button"
                  role="tab"
                  aria-selected={active}
                  onClick={() => chooseStatus(tab.status)}
                  className={`h-10 rounded-full px-4 text-sm font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent ${
                    active ? "bg-movo-card text-movo-accent" : "bg-movo-sand"
                  }`}
                >
                  {tab.label} ({count})
                </button>
              );
            })}
          </div>

          <label className="flex items-center gap-2 text-sm font-semibold">
            Categoría
            <select
              value={categoryId}
              onChange={(event) => {
                setCategoryId(Number(event.target.value));
                setPage(1);
              }}
              className="h-10 rounded-full bg-movo-sand px-4 outline-none focus:ring-2 focus:ring-movo-accent"
            >
              <option value={0}>Todas</option>
              {categories.status === "ready"
                ? categories.data.map((category) => (
                    <option key={category.id} value={category.id}>
                      {category.name}
                    </option>
                  ))
                : null}
            </select>
          </label>
        </div>
      </section>

      <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_26rem]">
        <section className="overflow-hidden rounded-3xl bg-white shadow-sm">
          {directory.state.status === "failed" ? (
            <div className="p-5">
              <Notice
                title="No pudimos cargar los locales"
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
                <th scope="col" className="hidden px-5 py-3 md:table-cell">
                  Categoría
                </th>
                <th scope="col" className="px-5 py-3">
                  Estado
                </th>
              </tr>
            </thead>
            <tbody>
              {data?.businesses.map((business) => (
                <tr
                  key={business.id}
                  className={`border-t border-movo-sand ${business.id === selected ? "bg-movo-accent/15" : ""}`}
                >
                  <td className="px-5 py-4">
                    <button
                      type="button"
                      onClick={() => setSelected(business.id)}
                      aria-pressed={business.id === selected}
                      className="text-left outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
                    >
                      <span className="block font-display text-base font-bold">
                        {business.name}
                      </span>
                      <span className="block text-xs text-movo-slate">
                        {business.owner_email} · #{business.id}
                      </span>
                    </button>
                  </td>
                  <td className="hidden px-5 py-4 md:table-cell">{business.category_name}</td>
                  <td className="px-5 py-4">
                    <StatusPill status={business.status} />
                    {business.pilot ? (
                      <span className="ml-2 text-xs font-bold text-movo-gold">Piloto</span>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          {data && data.businesses.length === 0 ? (
            <p className="p-8 text-center text-movo-slate">
              No hay locales que coincidan con los filtros.
            </p>
          ) : null}
          {directory.state.status === "loading" ? (
            <p className="p-8 text-center text-movo-slate">Cargando…</p>
          ) : null}

          {data && data.total > 0 ? (
            <nav
              aria-label="Páginas"
              className="flex flex-wrap items-center justify-between gap-3 border-t border-movo-sand px-5 py-4"
            >
              <p className="text-sm text-movo-slate">
                Mostrando {from}-{to} de {data.total} locales
              </p>
              <div className="flex items-center gap-1">
                <PageButton
                  disabled={page === 1}
                  onClick={() => setPage(page - 1)}
                  label="Anterior"
                >
                  ‹
                </PageButton>
                {pageWindow(page, pages).map((n) => (
                  <PageButton
                    key={n}
                    current={n === page}
                    onClick={() => setPage(n)}
                    label={`Página ${n}`}
                  >
                    {n}
                  </PageButton>
                ))}
                <PageButton
                  disabled={page >= pages}
                  onClick={() => setPage(page + 1)}
                  label="Siguiente"
                >
                  ›
                </PageButton>
              </div>
            </nav>
          ) : null}
        </section>

        {selected === null ? null : (
          <BusinessDetail
            // Cada local abre con su propio estado: sin esto se vería el anterior un instante.
            key={selected}
            id={selected}
            onClose={() => setSelected(null)}
            onChanged={directory.reload}
          />
        )}
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
