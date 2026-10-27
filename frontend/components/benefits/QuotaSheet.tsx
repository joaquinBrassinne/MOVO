"use client";

import { useEffect, useRef, useState, type SubmitEvent } from "react";

import { Notice } from "@/components/onboarding/fields";
import { CheckIcon, PlusIcon, XIcon } from "@/components/socios/icons";
import { setCouponQuota } from "@/lib/benefits/api";
import { parseQuotaForm } from "@/lib/benefits/schema";
import { QUOTA_SHORTCUTS, type CouponQuota } from "@/lib/benefits/types";
import { describeFailure, type FormMessage } from "@/lib/business/messages";

import { MinusIcon } from "./icons";
import { Switch } from "@/components/panel/Switch";

interface Props {
  quota: CouponQuota;
  onClose: () => void;
  /** Se llama después de guardar, antes de cerrar. */
  onSaved: () => void;
}

const DEFAULT_LIMIT = 30;

/** Hoja inferior para fijar el cupo mensual del local, o sacárselo. */
export function QuotaSheet({ quota, onClose, onSaved }: Props) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [unlimited, setUnlimited] = useState(quota.limit === null);
  const [text, setText] = useState(String(quota.limit ?? DEFAULT_LIMIT));
  const [error, setError] = useState<string | undefined>();
  const [notice, setNotice] = useState<FormMessage | null>(null);
  const [pending, setPending] = useState(false);

  // <dialog> da el foco atrapado, el cierre con Escape y el fondo oscuro sin código extra.
  useEffect(() => {
    const element = dialog.current;
    if (element && !element.open) element.showModal();
  }, []);

  const generated = quota.generated_this_month;
  const current = Number(text);
  const free = Number.isInteger(current) && current > 0 ? Math.max(0, current - generated) : null;

  function step(delta: number) {
    const base = Number.isInteger(current) && current > 0 ? current : DEFAULT_LIMIT;
    setText(String(Math.max(1, base + delta)));
    setError(undefined);
  }

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    setNotice(null);

    const parsed = parseQuotaForm(unlimited, text, generated);
    if (!parsed.ok) {
      setError(parsed.message);
      return;
    }
    setError(undefined);
    setPending(true);

    const result = await setCouponQuota(parsed.limit);
    setPending(false);
    if (!result.ok) {
      const feedback = describeFailure(result);
      setError(feedback.fields.limit);
      setNotice(feedback.form ?? null);
      return;
    }
    onSaved();
    onClose();
  }

  return (
    <dialog
      ref={dialog}
      onClose={onClose}
      aria-labelledby="quota-sheet-title"
      className="fixed inset-x-0 bottom-0 mx-auto mt-auto max-h-[92dvh] w-full max-w-md overflow-y-auto rounded-t-[2rem] bg-white p-0 text-movo-ink backdrop:bg-black/50"
    >
      <form onSubmit={submit} noValidate aria-busy={pending} className="space-y-5 p-6 pb-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 id="quota-sheet-title" className="font-display text-2xl font-extrabold">
              Cupo mensual
            </h2>
            <p className="mt-1 text-sm text-movo-slate">
              Los cupones que tus clientes pueden generar por mes en todo tu local.
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Cerrar"
            className="grid size-10 shrink-0 place-items-center rounded-full bg-movo-sand outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
          >
            <XIcon />
          </button>
        </div>

        <div className="flex items-center justify-between gap-4 rounded-2xl bg-movo-cream p-4">
          <div>
            <p className="font-semibold">Sin límite</p>
            <p className="text-sm text-movo-slate">
              Tus clientes pueden generar todos los que quieran.
            </p>
          </div>
          <Switch
            checked={unlimited}
            label="Sin límite de cupones"
            disabled={pending}
            onChange={(next) => {
              setUnlimited(next);
              setError(undefined);
            }}
          />
        </div>

        {unlimited ? null : (
          <>
            <div className="rounded-2xl bg-movo-cream p-4 text-center">
              <p className="text-xs font-bold tracking-wide text-movo-slate uppercase">
                Tope mensual
              </p>
              <div className="mt-2 flex items-center justify-between gap-3">
                <button
                  type="button"
                  aria-label="Restar un cupón"
                  disabled={pending}
                  onClick={() => step(-1)}
                  className="grid size-12 place-items-center rounded-full bg-white outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
                >
                  <MinusIcon />
                </button>
                <div>
                  <label htmlFor="quota-limit" className="sr-only">
                    Cupones por mes
                  </label>
                  <input
                    id="quota-limit"
                    inputMode="numeric"
                    value={text}
                    maxLength={6}
                    disabled={pending}
                    aria-invalid={error ? true : undefined}
                    aria-describedby={error ? "quota-limit-error" : undefined}
                    onChange={(event) => {
                      setText(event.target.value);
                      setError(undefined);
                    }}
                    className="w-28 bg-transparent text-center font-display text-5xl font-extrabold outline-none focus:ring-2 focus:ring-movo-accent"
                  />
                  <p className="text-sm text-movo-slate">cupones</p>
                </div>
                <button
                  type="button"
                  aria-label="Sumar un cupón"
                  disabled={pending}
                  onClick={() => step(1)}
                  className="grid size-12 place-items-center rounded-full bg-movo-accent outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
                >
                  <PlusIcon />
                </button>
              </div>
            </div>

            <div>
              <p className="mb-2 text-xs font-bold tracking-wide text-movo-slate uppercase">
                Accesos directos
              </p>
              <div className="grid grid-cols-4 gap-2">
                {QUOTA_SHORTCUTS.map((value) => (
                  <button
                    key={value}
                    type="button"
                    aria-pressed={current === value}
                    disabled={pending}
                    onClick={() => {
                      setText(String(value));
                      setError(undefined);
                    }}
                    className={`h-12 rounded-full font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink ${
                      current === value ? "bg-movo-accent" : "bg-movo-sand"
                    }`}
                  >
                    {value}
                  </button>
                ))}
              </div>
            </div>
          </>
        )}

        {error ? (
          <p id="quota-limit-error" role="alert" className="text-sm font-semibold text-movo-danger">
            {error}
          </p>
        ) : null}
        {notice ? <Notice title={notice.title} body={notice.body} /> : null}

        <p className="rounded-2xl bg-movo-sand p-4 text-sm text-movo-slate">
          Ya se {generated === 1 ? "generó" : "generaron"} <strong>{generated}</strong>{" "}
          {generated === 1 ? "cupón" : "cupones"} este mes
          {!unlimited && free !== null ? ` y quedarían ${free} libres` : ""}. El cupo no puede ser
          menor a lo ya generado y vuelve a empezar el primer día de cada mes.
        </p>

        <button
          type="submit"
          disabled={pending}
          className="flex h-14 w-full items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-60"
        >
          {pending ? "Guardando…" : "Guardar"}
          <CheckIcon />
        </button>
      </form>
    </dialog>
  );
}
