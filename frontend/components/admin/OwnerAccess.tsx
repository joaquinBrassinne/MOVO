"use client";

import { useState } from "react";

import { Notice } from "@/components/onboarding/fields";
import { issuePasswordReset } from "@/lib/passwordreset/api";
import {
  RESET_VALID_HOURS,
  buildResetLink,
  describeIssueFailure,
  resetWhatsAppUrl,
  type Message,
} from "@/lib/passwordreset/rules";

interface Props {
  email: string;
  /** Owner WhatsApp (digits) from the business detail; null hides the WhatsApp shortcut. */
  whatsapp: string | null;
}

type Step =
  | { name: "idle" }
  | { name: "confirming" }
  | { name: "issuing" }
  | { name: "issued"; link: string };

/** "Acceso del dueño": assisted password reset. Render with `key={businessId}` to reset the flow. */
export function OwnerAccess({ email, whatsapp }: Props) {
  const [step, setStep] = useState<Step>({ name: "idle" });
  const [message, setMessage] = useState<Message | null>(null);
  const [copied, setCopied] = useState(false);

  async function issue() {
    setStep({ name: "issuing" });
    setMessage(null);

    const result = await issuePasswordReset(email);
    if (!result.ok) {
      setMessage(describeIssueFailure(result));
      setStep({ name: "idle" });
      return;
    }
    setStep({ name: "issued", link: buildResetLink(window.location.origin, result.data.token) });
  }

  async function copy(link: string) {
    try {
      await navigator.clipboard.writeText(link);
      setCopied(true);
    } catch {
      setMessage({ title: "No pudimos copiar", body: "Seleccioná el enlace y copialo a mano." });
    }
  }

  const whatsappUrl =
    step.name === "issued" && whatsapp ? resetWhatsAppUrl(whatsapp, step.link) : null;

  return (
    <section className="space-y-3">
      <h3 className="text-xs font-bold tracking-wide text-movo-slate uppercase">
        Acceso del dueño
      </h3>
      <div className="space-y-3 rounded-2xl bg-movo-cream p-4">
        <p className="text-xs font-bold tracking-wide text-movo-slate uppercase">
          Email de ingreso del dueño
        </p>
        <p className="rounded-xl bg-white px-4 py-3 text-sm font-semibold break-all">{email}</p>

        {message ? <Notice title={message.title} body={message.body} /> : null}

        {step.name === "idle" || step.name === "issuing" ? (
          <>
            <button
              type="button"
              disabled={step.name === "issuing"}
              onClick={() => {
                setCopied(false);
                setStep({ name: "confirming" });
              }}
              className="h-12 w-full rounded-full bg-white font-display font-bold text-movo-ink ring-1 ring-movo-slate/30 outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-60"
            >
              Generar enlace de reseteo
            </button>
            <p className="text-center text-sm text-movo-slate">
              Permite al dueño restablecer su clave de forma asistida.
            </p>
          </>
        ) : null}

        {step.name === "confirming" ? (
          <div role="alertdialog" aria-label="Confirmar el reseteo" className="space-y-3">
            <p className="text-sm font-semibold">
              Se va a generar un enlace nuevo y se cerrarán todas las sesiones del dueño. El enlace
              anterior, si existía, deja de valer.
            </p>
            <div className="flex gap-3">
              <button
                type="button"
                onClick={() => void issue()}
                className="h-11 flex-1 rounded-full bg-movo-accent font-display font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
              >
                Sí, generar
              </button>
              <button
                type="button"
                onClick={() => setStep({ name: "idle" })}
                className="h-11 flex-1 rounded-full bg-white font-bold ring-1 ring-movo-slate/30 outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
              >
                Cancelar
              </button>
            </div>
          </div>
        ) : null}

        {step.name === "issued" ? (
          <div className="space-y-3">
            <p className="text-sm font-semibold">
              Enlace listo. Vale {RESET_VALID_HOURS} horas y se puede usar una sola vez; se muestra
              solo ahora.
            </p>
            <input
              readOnly
              aria-label="Enlace de reseteo"
              value={step.link}
              onFocus={(event) => event.currentTarget.select()}
              className="w-full rounded-xl bg-white px-3 py-2 text-xs"
            />
            <div className="flex flex-wrap gap-3">
              <button
                type="button"
                onClick={() => void copy(step.link)}
                className="rounded-full bg-movo-accent px-4 py-2 text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
              >
                {copied ? "Enlace copiado" : "Copiar enlace"}
              </button>
              {whatsappUrl ? (
                <a
                  href={whatsappUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="rounded-full bg-movo-ok/15 px-4 py-2 text-sm font-bold text-movo-ok"
                >
                  Enviar por WhatsApp
                </a>
              ) : null}
            </div>
          </div>
        ) : null}
      </div>
    </section>
  );
}
