import Link from "next/link";

import { Field, TextInput } from "@/components/onboarding/fields";
import { LockIcon } from "@/components/socios/icons";

interface EditableProps {
  value: string;
  onChange: (value: string) => void;
  error?: string;
  disabled?: boolean;
}

/** Ticket promedio for a plan with the advanced metrics: a clearable amount in whole pesos. */
export function TicketField({ value, onChange, error, disabled }: EditableProps) {
  return (
    <Field
      id="average_ticket"
      label="Ticket promedio (opcional)"
      error={error}
      help="Lo usamos para estimar tu retorno económico en Métricas. Es confidencial y solo lo ves vos."
    >
      <div className="flex gap-3">
        <span className="flex h-14 shrink-0 items-center rounded-2xl bg-movo-sand px-4 text-base font-semibold">
          $
        </span>
        <TextInput
          id="average_ticket"
          inputMode="numeric"
          value={value}
          onChange={onChange}
          error={error}
          disabled={disabled}
          placeholder="15000"
          maxLength={12}
        />
      </div>
    </Field>
  );
}

/** Ticket promedio for a plan without the advanced metrics: shown, but locked. */
export function TicketLockedCard() {
  return (
    <section className="space-y-4 rounded-3xl bg-movo-sand/70 p-5" aria-label="Ticket promedio">
      <div className="flex items-start gap-3">
        <span className="flex size-10 shrink-0 items-center justify-center rounded-full bg-white">
          <LockIcon className="size-5" />
        </span>
        <div className="min-w-0 flex-1">
          <h3 className="font-display text-lg leading-tight font-extrabold">
            Ticket promedio (opcional)
          </h3>
          <p className="text-sm text-movo-slate">Estimación interna del valor por cliente.</p>
        </div>
        <span className="rounded-full bg-white px-2 py-1 text-xs font-bold tracking-wider text-movo-slate">
          BLOQUEADO
        </span>
      </div>

      <input
        type="text"
        disabled
        value="••••••"
        aria-label="Ticket promedio, bloqueado"
        className="h-14 w-full rounded-2xl bg-white/60 px-4 text-base text-movo-slate"
      />

      <div className="space-y-3 rounded-2xl bg-white p-4">
        <p className="text-sm font-bold">Disponible en el Plan Pro</p>
        <p className="text-sm text-movo-slate">
          Lo usamos para estimar tu retorno automático en el panel de Métricas. Solo lo ves vos.
        </p>
        <Link
          href="/owners/plan"
          className="inline-flex h-11 items-center rounded-full bg-movo-accent px-5 text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
        >
          Ver Plan Pro
        </Link>
      </div>
    </section>
  );
}
