import Link from "next/link";

import { Logo } from "@/components/Logo";
import type { Plan } from "@/lib/plans/types";

import { PlanCard } from "./PlanCard";

const CATEGORIES = ["Gimnasios", "Pilates", "Yoga", "Estudios", "Complejos deportivos"];

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

/** Answers restate what the page already promises; no new commitments. */
const FAQ = [
  {
    question: "¿Pago comisión por cada reserva?",
    answer:
      "No. Pagás solo el plan mensual. Las reservas se coordinan directo con vos por WhatsApp.",
  },
  {
    question: "¿Cómo se cobra el plan?",
    answer: "Contratás con el link de Mercado Pago y el cobro mensual es automático.",
  },
  {
    question: "¿Cuándo aparece mi negocio en el mapa?",
    answer:
      "Cuando completás los datos de tu negocio, el equipo de MOVO revisa tu perfil y lo publica en el mapa y en la app.",
  },
];

const FOCUS =
  "focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-movo-accent";

export function PlansLanding({ plans }: { plans: Plan[] | null }) {
  return (
    <div className="min-h-screen overflow-x-clip bg-movo-tabs text-white">
      <header>
        <nav
          aria-label="Principal"
          className="mx-auto flex h-20 max-w-6xl items-center justify-between gap-4 px-4 sm:px-8"
        >
          <Link href="/" aria-label="MOVO, inicio" className={FOCUS}>
            <Logo className="h-7" />
          </Link>
          <Link
            href="/owners/sign-in"
            className={`rounded-full bg-white px-4 py-2.5 text-sm font-semibold text-movo-ink hover:bg-movo-sand ${FOCUS}`}
          >
            Ingreso Dueños
          </Link>
        </nav>
      </header>

      <main>
        <section className="relative mx-auto max-w-6xl px-4 pt-14 pb-20 sm:px-8 sm:pt-22 sm:pb-24">
          <div
            aria-hidden="true"
            className="pointer-events-none absolute -top-10 -right-32 size-[620px] rounded-full bg-radial from-movo-accent/15 to-transparent to-65%"
          />
          <h1 className="relative max-w-[11ch] font-display text-5xl leading-[0.95] font-extrabold tracking-tight sm:text-7xl lg:text-8xl">
            Llená tus clases con gente de <span className="text-movo-accent">tu zona.</span>
          </h1>
          <div className="relative mt-10 flex flex-col gap-8 sm:mt-12 sm:flex-row sm:items-end sm:justify-between">
            <p className="max-w-[44ch] text-lg leading-relaxed text-movo-muted sm:text-xl">
              Sumá tu gimnasio, estudio o complejo a la red MOVO. Publicá tus beneficios exclusivos
              y atraé nuevos alumnos sin pagar comisiones por reserva.
            </p>
            <a
              href="#planes"
              className={`w-fit shrink-0 rounded-full bg-movo-accent px-7 py-4 font-bold text-movo-ink transition hover:brightness-95 motion-reduce:transition-none ${FOCUS}`}
            >
              Ver planes
            </a>
          </div>
        </section>

        <ul
          aria-label="Para todo tipo de espacios"
          className="flex flex-wrap justify-center gap-x-12 gap-y-3 border-y border-movo-muted/20 px-4 py-5 font-semibold"
        >
          {CATEGORIES.map((category) => (
            <li key={category}>{category}</li>
          ))}
        </ul>

        <section
          id="planes"
          aria-labelledby="planes-title"
          className="mx-auto max-w-6xl scroll-mt-4 px-4 py-20 sm:px-8 sm:py-28"
        >
          <h2
            id="planes-title"
            className="text-center font-display text-4xl font-extrabold tracking-tight sm:text-5xl"
          >
            Planes para tu negocio
          </h2>
          <p className="mt-3 text-center text-movo-muted">
            Elegí la opción que mejor se adapte al tamaño de tu espacio.
          </p>

          {plans && plans.length > 0 ? (
            <ul className="mx-auto mt-14 grid max-w-4xl gap-5 md:grid-cols-2">
              {plans.map((plan) => (
                <PlanCard key={plan.plan} plan={plan} featured={plan.plan === "pro"} />
              ))}
            </ul>
          ) : (
            <p
              role="status"
              className="mx-auto mt-14 max-w-md rounded-3xl bg-movo-card p-6 text-center text-movo-muted ring-1 ring-movo-muted/15"
            >
              No pudimos cargar los planes en este momento. Probá de nuevo en unos minutos.
            </p>
          )}
        </section>

        <section
          aria-labelledby="proceso-title"
          className="mx-auto max-w-6xl px-4 pb-20 sm:px-8 sm:pb-28"
        >
          <h2
            id="proceso-title"
            className="font-display text-3xl font-extrabold tracking-tight sm:text-4xl"
          >
            Proceso simple y transparente en 3 pasos
          </h2>
          <ol className="mt-8 grid gap-5 md:grid-cols-3">
            {STEPS.map((step, index) => (
              <li
                key={step.title}
                className="flex min-h-60 flex-col justify-between gap-8 rounded-3xl bg-movo-card p-8"
              >
                <span
                  aria-hidden="true"
                  className="font-display text-6xl leading-none font-extrabold text-movo-accent"
                >
                  {index + 1}
                </span>
                <div>
                  <h3 className="text-xl font-bold">{step.title}</h3>
                  <p className="mt-2 leading-relaxed text-movo-muted">{step.body}</p>
                </div>
              </li>
            ))}
          </ol>
        </section>

        <section
          aria-labelledby="faq-title"
          className="mx-auto grid max-w-6xl gap-8 px-4 pb-20 sm:px-8 sm:pb-28 md:grid-cols-3 md:gap-12"
        >
          <h2
            id="faq-title"
            className="font-display text-3xl font-extrabold tracking-tight sm:text-4xl"
          >
            Preguntas frecuentes
          </h2>
          <div className="md:col-span-2">
            {FAQ.map((item) => (
              <details key={item.question} className="group border-b border-movo-muted/20 py-5">
                <summary
                  className={`flex cursor-pointer list-none items-center justify-between gap-4 rounded text-lg font-semibold [&::-webkit-details-marker]:hidden ${FOCUS}`}
                >
                  {item.question}
                  <span
                    aria-hidden="true"
                    className="text-2xl leading-none text-movo-accent group-open:rotate-45 motion-safe:transition-transform"
                  >
                    +
                  </span>
                </summary>
                <p className="mt-3 leading-relaxed text-movo-muted">{item.answer}</p>
              </details>
            ))}
          </div>
        </section>
      </main>

      <footer className="border-t border-movo-muted/20">
        <div className="mx-auto flex max-w-6xl flex-col gap-3 px-4 py-8 text-sm text-movo-muted sm:flex-row sm:justify-between sm:px-8">
          <span>© MOVO. Todos los derechos reservados.</span>
          <Link href="/owners/sign-in" className={`hover:text-white ${FOCUS}`}>
            Ingreso Dueños
          </Link>
        </div>
      </footer>
    </div>
  );
}
