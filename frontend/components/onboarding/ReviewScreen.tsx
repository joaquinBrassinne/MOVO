"use client";

import Link from "next/link";
import type { ReactNode } from "react";

import { ArrowRightIcon, ClockIcon, MapPinIcon, StoreIcon } from "@/components/socios/icons";
import { useServices } from "@/lib/business/hooks";
import { fromWhatsApp } from "@/lib/business/phone";
import type { MyBusiness } from "@/lib/business/types";
import { PANEL_PATH } from "@/lib/session/hooks";

/** Pantalla de espera: el negocio ya se envió y el equipo de MOVO lo revisa a mano. */
export function ReviewScreen({ data }: { data: MyBusiness }) {
  const { business, branches } = data;
  const services = useServices();
  const branch = branches[0];
  const serviceCount = services.state.status === "ready" ? services.state.data.length : null;

  return (
    <div className="relative isolate flex min-h-dvh flex-col overflow-hidden bg-movo-page font-sans text-white">
      <Eclipse />

      <main className="mx-auto flex w-full max-w-md flex-1 flex-col px-5 pt-16 pb-8">
        <p className="mx-auto inline-flex items-center gap-2 rounded-full bg-movo-card px-4 py-2 text-sm font-semibold">
          <ClockIcon className="size-4 text-movo-accent" />
          En revisión · Tiempo estimado &lt; 2 hs
        </p>

        <h1 className="mt-8 text-center font-display text-4xl leading-tight font-extrabold">
          Tu negocio está en revisión
        </h1>
        <p className="mt-4 text-center text-lg leading-relaxed text-movo-muted">
          Estamos confirmando tu pago y revisando tus datos. Te avisamos por WhatsApp apenas esté
          listo.
        </p>

        <section className="mt-8 space-y-5 rounded-3xl bg-movo-card p-6">
          <div className="flex items-center gap-4">
            <span className="grid size-14 shrink-0 place-items-center rounded-full bg-movo-field font-display text-xl font-bold text-movo-accent">
              {business.name.slice(0, 1).toUpperCase()}
            </span>
            <p className="min-w-0 flex-1 truncate font-display text-xl font-bold">
              {business.name}
            </p>
            <span className="shrink-0 rounded-full bg-movo-accent px-3 py-1 text-xs font-bold text-movo-ink">
              En revisión
            </span>
          </div>

          <Detail icon={<MapPinIcon className="size-5" />} label="Sede física">
            {branch ? `${branch.address}, ${branch.city}` : "Todavía no cargaste una sede"}
          </Detail>
          <Detail icon={<StoreIcon className="size-5" />} label="Servicios cargados">
            {serviceCount === null
              ? "…"
              : serviceCount === 0
                ? "Ninguno por ahora"
                : `${serviceCount} ${serviceCount === 1 ? "servicio" : "servicios"}`}
          </Detail>
          <Detail icon={<span className="font-bold">WA</span>} label="WhatsApp de contacto">
            +54 {fromWhatsApp(business.whatsapp)}
          </Detail>
        </section>

        <p className="mt-5 rounded-3xl bg-movo-card/60 p-5 text-sm leading-relaxed text-movo-muted">
          Cuando esté aprobado te escribimos al número de WhatsApp que informaste, con el acceso
          directo a tu panel.
        </p>

        <Link
          href={PANEL_PATH}
          className="mt-auto flex h-16 items-center justify-center gap-3 rounded-full bg-movo-accent font-display text-xl font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-white"
        >
          Ver mi panel
          <ArrowRightIcon />
        </Link>
      </main>
    </div>
  );
}

function Detail({
  icon,
  label,
  children,
}: {
  icon: ReactNode;
  label: string;
  children: ReactNode;
}) {
  return (
    <div className="flex items-start gap-4">
      <span className="grid size-10 shrink-0 place-items-center rounded-full bg-movo-field text-movo-accent">
        {icon}
      </span>
      <div className="min-w-0">
        <p className="text-xs font-bold tracking-wide text-movo-muted uppercase">{label}</p>
        <p className="mt-0.5 font-semibold break-words">{children}</p>
      </div>
    </div>
  );
}

/** El sol y los anillos de fondo: puro adorno. */
function Eclipse() {
  return (
    <div aria-hidden="true" className="pointer-events-none absolute inset-x-0 top-0 -z-10 h-72">
      <div className="absolute top-8 left-1/2 size-56 -translate-x-1/2 rounded-full border border-dashed border-movo-accent/40" />
      <div className="absolute top-16 left-1/2 size-40 -translate-x-1/2 rounded-full bg-movo-accent shadow-[0_0_80px_rgb(252_211_77/0.45)]" />
    </div>
  );
}
