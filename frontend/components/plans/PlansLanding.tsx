import Link from "next/link";

import { Logo } from "@/components/Logo";
import type { Plan } from "@/lib/plans/types";

import { PlanCard } from "./PlanCard";

const STEPS = [
  {
    title: "Elegí tu plan",
    body: "Seleccioná Starter o Pro según la cantidad de sedes y la visibilidad que necesitás.",
  },
  {
    title: "Pagá con Mercado Pago",
    body: "Contratás con el link de Mercado Pago y el cobro mensual es automático. Después completás los datos de tu negocio.",
  },
  {
    title: "Revisamos y publicamos",
    body: "El equipo de MOVO revisa tu perfil y lo publica en el mapa y en la app.",
  },
];

const FOCUS =
  "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-movo-accent";

export function PlansLanding({ plans }: { plans: Plan[] | null }) {
  return (
    <div className="min-h-screen bg-movo-cream text-slate-900">
      <header className="bg-movo-page">
        <nav
          aria-label="Principal"
          className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-4"
        >
          <Link href="/" aria-label="MOVO, inicio" className={FOCUS}>
            <Logo className="h-7" />
          </Link>
          <Link
            href="/socios/ingresar"
            className={`rounded-full border border-white/30 px-4 py-2 text-sm font-semibold text-white hover:bg-white/10 ${FOCUS}`}
          >
            Ingreso Dueños
          </Link>
        </nav>
      </header>

      <main>
        <section className="bg-movo-page px-4 pt-10 pb-16 text-center text-white sm:pt-16 sm:pb-24">
          <div className="mx-auto max-w-3xl">
            <h1 className="font-display text-4xl font-extrabold tracking-tight sm:text-6xl">
              Llená tus clases con gente de tu zona
            </h1>
            <p className="mx-auto mt-5 max-w-xl text-movo-muted sm:text-lg">
              Sumá tu gimnasio, estudio o complejo a la red MOVO. Publicá tus beneficios exclusivos
              y atraé nuevos alumnos sin pagar comisiones por reserva.
            </p>
            <a
              href="#planes"
              className={`mt-8 inline-block rounded-full bg-movo-accent px-6 py-3 text-sm font-bold text-movo-ink hover:brightness-95 ${FOCUS}`}
            >
              Ver planes ↓
            </a>
          </div>
        </section>

        <section id="planes" aria-labelledby="planes-title" className="px-4 py-14 sm:py-20">
          <div className="mx-auto max-w-5xl">
            <h2
              id="planes-title"
              className="font-display text-center text-3xl font-extrabold text-movo-ink"
            >
              Planes para tu negocio
            </h2>
            <p className="mx-auto mt-3 max-w-xl text-center text-movo-slate">
              Elegí la opción que mejor se adapte al tamaño de tu espacio.
            </p>

            {plans && plans.length > 0 ? (
              <ul className="mx-auto mt-12 grid max-w-3xl gap-8 md:grid-cols-2">
                {plans.map((plan) => (
                  <PlanCard key={plan.plan} plan={plan} featured={plan.plan === "pro"} />
                ))}
              </ul>
            ) : (
              <p
                role="status"
                className="mx-auto mt-12 max-w-md rounded-2xl bg-white p-6 text-center text-movo-slate ring-1 ring-slate-200"
              >
                No pudimos cargar los planes en este momento. Probá de nuevo en unos minutos.
              </p>
            )}
          </div>
        </section>

        <section aria-labelledby="proceso-title" className="px-4 pb-16">
          <div className="mx-auto max-w-4xl rounded-3xl bg-white p-6 ring-1 ring-slate-200 sm:p-10">
            <h2
              id="proceso-title"
              className="text-center text-xs font-bold tracking-widest text-movo-ink uppercase"
            >
              Proceso simple y transparente en 3 pasos
            </h2>
            <ol className="mt-8 grid gap-8 sm:grid-cols-3">
              {STEPS.map((step, index) => (
                <li key={step.title} className="text-center">
                  <span
                    aria-hidden="true"
                    className="mx-auto flex size-10 items-center justify-center rounded-full bg-movo-page font-bold text-movo-accent"
                  >
                    {index + 1}
                  </span>
                  <h3 className="mt-3 text-sm font-bold text-movo-ink">{step.title}</h3>
                  <p className="mt-1 text-sm text-movo-slate">{step.body}</p>
                </li>
              ))}
            </ol>
          </div>
        </section>
      </main>

      <footer className="bg-movo-page px-4 py-8 text-center text-sm text-movo-muted">
        © MOVO. Todos los derechos reservados.
      </footer>
    </div>
  );
}
