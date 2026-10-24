import type { ReactNode } from "react";

import { ArrowLeftIcon, ArrowRightIcon, UserIcon } from "@/components/socios/icons";

export const STEPS = [
  { number: 1, label: "Negocio" },
  { number: 2, label: "Sede" },
  { number: 3, label: "Servicios" },
  { number: 4, label: "Fotos" },
] as const;

export type StepNumber = (typeof STEPS)[number]["number"];

interface ShellProps {
  step: StepNumber;
  children: ReactNode;
}

/** Marco de los cuatro pasos: encabezado oscuro, progreso y el contenido del paso. */
export function Shell({ step, children }: ShellProps) {
  const current = STEPS[step - 1];

  return (
    <div className="flex min-h-dvh flex-col bg-movo-cream font-sans text-movo-ink">
      <header className="sticky top-0 z-20 bg-movo-page text-white">
        <div className="mx-auto flex h-20 w-full max-w-md items-center justify-between px-5">
          <span className="font-display text-2xl font-extrabold tracking-wider text-movo-accent">
            MOVO
          </span>
          <h1 className="font-display text-lg font-bold">
            Paso {step} De {STEPS.length}: {current.label}
          </h1>
          <span
            className="grid size-10 place-items-center rounded-full bg-movo-card"
            aria-hidden="true"
          >
            <UserIcon />
          </span>
        </div>
      </header>

      <main className="mx-auto w-full max-w-md flex-1 space-y-5 px-4 pt-5 pb-32">
        <Progress step={step} />
        {children}
      </main>
    </div>
  );
}

function Progress({ step }: { step: StepNumber }) {
  return (
    <nav aria-label="Progreso del alta" className="rounded-3xl bg-white p-5 shadow-sm">
      <div className="flex items-center justify-between">
        <p className="font-display text-xl font-bold">
          <span className="mr-2 inline-grid size-7 place-items-center rounded-full bg-movo-accent text-sm">
            {step}
          </span>
          Paso {step} de {STEPS.length}
        </p>
        <p className="text-sm font-semibold text-movo-slate">{STEPS[step - 1].label}</p>
      </div>

      <ol className="mt-4 grid grid-cols-4 gap-2">
        {STEPS.map(({ number, label }) => (
          <li key={number} aria-current={number === step ? "step" : undefined}>
            <span
              className={`block h-1.5 rounded-full ${number <= step ? "bg-movo-accent" : "bg-movo-sand"}`}
            />
            <span
              className={`mt-2 block text-xs ${number === step ? "font-bold" : "text-movo-slate"}`}
            >
              {label}
            </span>
          </li>
        ))}
      </ol>
    </nav>
  );
}

interface FooterProps {
  onBack?: () => void;
  /** Texto del botón principal. */
  nextLabel?: string;
  pending?: boolean;
  disabled?: boolean;
}

/** Barra inferior fija con "Atrás" y el botón principal, que envía el formulario `form`. */
export function StepFooter({ onBack, nextLabel = "Siguiente", pending, disabled }: FooterProps) {
  return (
    <div className="fixed inset-x-0 bottom-0 z-20 bg-movo-page px-4 py-4">
      <div className="mx-auto flex w-full max-w-md items-center gap-3">
        <button
          type="button"
          onClick={onBack}
          disabled={!onBack || pending}
          className="flex h-14 flex-1 items-center justify-center gap-2 rounded-full border border-white/30 font-display font-bold text-white outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-40"
        >
          <ArrowLeftIcon />
          Atrás
        </button>
        <button
          type="submit"
          form="step-form"
          disabled={pending || disabled}
          className="flex h-14 flex-[1.4] items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-white disabled:opacity-60"
        >
          {pending ? "Guardando…" : nextLabel}
          {pending ? null : <ArrowRightIcon />}
        </button>
      </div>
    </div>
  );
}
