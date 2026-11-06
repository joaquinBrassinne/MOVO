"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";

import { AuthField } from "@/components/socios/AuthField";
import { FormAlert } from "@/components/socios/FormAlert";
import { CheckIcon } from "@/components/socios/icons";
import { useResetPassword } from "@/lib/passwordreset/hooks";
import { PASSWORD_MIN_LENGTH } from "@/lib/session/messages";

interface Props {
  /** WhatsApp link of the support; null when no number is configured. */
  supportUrl: string | null;
}

export function ResetPasswordForm({ supportUrl }: Props) {
  const token = useSearchParams().get("token");
  const reset = useResetPassword(token && token.trim() !== "" ? token : null);

  if (reset.status === "done") {
    return (
      <Card>
        <h2 className="font-display text-2xl font-extrabold">Clave guardada</h2>
        <p className="mt-2 text-movo-slate">Ya podés ingresar a tu panel con la clave nueva.</p>
        <Link
          href="/owners/sign-in"
          className="mt-6 flex h-14 w-full items-center justify-center rounded-full bg-movo-accent font-display text-lg font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
        >
          Ir a ingresar
        </Link>
      </Card>
    );
  }

  if (reset.status === "invalid") {
    return (
      <Card>
        <h2 className="font-display text-2xl font-extrabold">Enlace no válido</h2>
        <p className="mt-2 text-movo-slate">
          El enlace venció o ya se usó. Pedile uno nuevo a soporte de MOVO.
        </p>
        {supportUrl ? (
          <a
            href={supportUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-6 flex h-14 w-full items-center justify-center rounded-full bg-movo-accent font-display text-lg font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
          >
            Hablar con soporte
          </a>
        ) : null}
      </Card>
    );
  }

  const { checks } = reset;
  return (
    <>
      <Card>
        <form onSubmit={reset.submit} noValidate aria-busy={reset.pending} className="space-y-5">
          <AuthField
            id="new-password"
            label="Contraseña nueva"
            tone="light"
            type="password"
            autoComplete="new-password"
            value={reset.password}
            onChange={reset.setPassword}
            disabled={reset.pending}
          />
          <AuthField
            id="repeat-password"
            label="Repetir contraseña"
            tone="light"
            type="password"
            autoComplete="new-password"
            value={reset.repeat}
            onChange={reset.setRepeat}
            disabled={reset.pending}
          />

          <ul className="space-y-2 rounded-3xl bg-stone-100 p-4 text-sm">
            <Rule ok={checks.longEnough}>Mínimo {PASSWORD_MIN_LENGTH} caracteres</Rule>
            <Rule ok={checks.matches}>Las dos coinciden</Rule>
          </ul>

          {reset.message ? (
            <FormAlert title={reset.message.title} body={reset.message.body} />
          ) : null}

          <button
            type="submit"
            disabled={reset.pending || !checks.valid}
            className="h-14 w-full rounded-full bg-movo-accent font-display text-lg font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-60"
          >
            {reset.pending ? "Guardando…" : "Guardar clave"}
          </button>
        </form>
      </Card>
      {supportUrl ? (
        <a
          href={supportUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-6 block text-center text-sm font-semibold text-movo-muted hover:text-movo-accent"
        >
          ¿Tenés problemas? Hablá con soporte de MOVO
        </a>
      ) : null}
    </>
  );
}

function Card({ children }: { children: React.ReactNode }) {
  return <section className="rounded-[28px] bg-white p-6 text-movo-ink">{children}</section>;
}

function Rule({ ok, children }: { ok: boolean; children: React.ReactNode }) {
  return (
    <li className="flex items-center gap-3">
      <span
        className={`grid size-6 place-items-center rounded-full ${ok ? "bg-movo-ok text-white" : "bg-stone-200 text-transparent"}`}
        aria-hidden="true"
      >
        <CheckIcon className="size-4" />
      </span>
      <span>
        {children}
        <span className="sr-only">{ok ? " (cumple)" : " (pendiente)"}</span>
      </span>
    </li>
  );
}
