"use client";

import { useEffect, useRef, useState, type ReactNode, type SubmitEvent } from "react";

import { Field, Notice, TextArea, TextInput } from "@/components/onboarding/fields";
import { Switch } from "@/components/panel/Switch";
import { CheckIcon, XIcon } from "@/components/socios/icons";
import { addBenefit, editBenefit, setBenefitActive } from "@/lib/benefits/api";
import { parseBenefitForm, type BenefitFormValues } from "@/lib/benefits/schema";
import {
  BENEFIT_TYPE,
  MAX_CONDITIONS_LENGTH,
  MAX_TITLE_LENGTH,
  PERCENT_SHORTCUTS,
  type BenefitListItem,
  type BenefitType,
} from "@/lib/benefits/types";
import { describeFailure, type FormMessage } from "@/lib/business/messages";

import { DollarIcon, GiftIcon, PercentIcon } from "./icons";

interface Props {
  /** El beneficio a editar, o `null` para crear uno nuevo. */
  benefit: BenefitListItem | null;
  onClose: () => void;
  /** Se llama después de guardar, antes de cerrar. */
  onSaved: () => void;
}

const TYPES = [
  { type: BENEFIT_TYPE.PERCENT, label: "Porcentaje", icon: PercentIcon },
  { type: BENEFIT_TYPE.AMOUNT, label: "Monto fijo", icon: DollarIcon },
  { type: BENEFIT_TYPE.CUSTOM, label: "Otro beneficio", icon: GiftIcon },
] as const;

/** El nombre con el que la API llama a cada campo del formulario. */
const API_FIELD: Record<keyof BenefitFormValues, string> = {
  type: "type",
  percent: "percent",
  amount: "amount_ars",
  title: "title",
  conditions: "conditions",
};

function initialValues(benefit: BenefitListItem | null): BenefitFormValues {
  return {
    type: benefit?.type ?? BENEFIT_TYPE.PERCENT,
    percent: benefit?.percent ? String(benefit.percent) : "",
    amount: benefit?.amount_ars ? String(benefit.amount_ars) : "",
    title: benefit?.title ?? "",
    conditions: benefit?.conditions ?? "",
  };
}

/** Hoja inferior para crear o editar un beneficio: porcentaje, monto fijo u otro beneficio. */
export function BenefitSheet({ benefit, onClose, onSaved }: Props) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [values, setValues] = useState<BenefitFormValues>(() => initialValues(benefit));
  const [active, setActive] = useState(benefit?.active ?? true);
  const [fields, setFields] = useState<Record<string, string>>({});
  const [notice, setNotice] = useState<FormMessage | null>(null);
  const [pending, setPending] = useState(false);

  // <dialog> da el foco atrapado, el cierre con Escape y el fondo oscuro sin código extra.
  useEffect(() => {
    const element = dialog.current;
    if (element && !element.open) element.showModal();
  }, []);

  function set<K extends keyof BenefitFormValues>(key: K, value: BenefitFormValues[K]) {
    setValues((current) => ({ ...current, [key]: value }));
    // Escribir en un campo borra su error: el aviso era sobre lo que había antes.
    setFields((current) => {
      const field = API_FIELD[key];
      if (!(field in current)) return current;
      const next = { ...current };
      delete next[field];
      return next;
    });
  }

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    setNotice(null);

    const parsed = parseBenefitForm(values);
    if (!parsed.ok) {
      setFields(parsed.fields);
      return;
    }
    setFields({});
    setPending(true);

    const saved = benefit
      ? await editBenefit(benefit.id, parsed.input)
      : await addBenefit(parsed.input);
    if (!saved.ok) {
      const feedback = describeFailure(saved);
      setFields(feedback.fields);
      setNotice(feedback.form ?? null);
      setPending(false);
      return;
    }

    // El interruptor del dueño se guarda aparte: crear y editar no lo cambian.
    const switched =
      saved.data.active === active ? null : await setBenefitActive(saved.data.id, active);
    setPending(false);
    if (switched && !switched.ok) {
      // El beneficio ya quedó guardado: se avisa del interruptor y se recarga la lista.
      onSaved();
      setNotice(describeFailure(switched).form ?? null);
      return;
    }
    onSaved();
    onClose();
  }

  const percent = Number(values.percent);

  return (
    <dialog
      ref={dialog}
      onClose={onClose}
      aria-labelledby="benefit-sheet-title"
      className="fixed inset-x-0 bottom-0 mx-auto mt-auto max-h-[92dvh] w-full max-w-md overflow-y-auto rounded-t-[2rem] bg-white p-0 text-movo-ink backdrop:bg-black/50"
    >
      <form onSubmit={submit} noValidate aria-busy={pending} className="space-y-5 p-6 pb-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 id="benefit-sheet-title" className="font-display text-2xl font-extrabold">
              {benefit ? "Editar beneficio" : "Nuevo beneficio"}
            </h2>
            <p className="mt-1 text-sm text-movo-slate">
              Configurá una promoción para atraer nuevos alumnos.
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

        <fieldset>
          <legend className="mb-2 text-base font-semibold">Tipo de beneficio</legend>
          <div className="grid grid-cols-3 gap-2">
            {TYPES.map(({ type, label, icon: Icon }) => (
              <TypeButton
                key={type}
                selected={values.type === type}
                disabled={pending}
                onSelect={() => set("type", type as BenefitType)}
              >
                <Icon className="size-5" />
                {label}
              </TypeButton>
            ))}
          </div>
        </fieldset>

        {values.type === BENEFIT_TYPE.PERCENT ? (
          <Field id="benefit-percent" label="Porcentaje de descuento" error={fields.percent}>
            <div className="mb-2 grid grid-cols-3 gap-2" role="group" aria-label="Atajos">
              {PERCENT_SHORTCUTS.map((shortcut) => (
                <button
                  key={shortcut}
                  type="button"
                  aria-pressed={percent === shortcut}
                  disabled={pending}
                  onClick={() => set("percent", String(shortcut))}
                  className={`h-11 rounded-full font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink ${
                    percent === shortcut ? "bg-movo-accent" : "bg-movo-sand"
                  }`}
                >
                  {shortcut}%
                </button>
              ))}
            </div>
            <TextInput
              id="benefit-percent"
              value={values.percent}
              onChange={(value) => set("percent", value)}
              error={fields.percent}
              disabled={pending}
              inputMode="numeric"
              maxLength={3}
              placeholder="Otro porcentaje, de 1 a 100"
            />
          </Field>
        ) : null}

        {values.type === BENEFIT_TYPE.AMOUNT ? (
          <Field id="benefit-amount" label="Monto de descuento" error={fields.amount_ars}>
            <div className="relative">
              <span
                aria-hidden="true"
                className="pointer-events-none absolute top-1/2 left-4 -translate-y-1/2 font-bold text-movo-slate"
              >
                $
              </span>
              <TextInput
                id="benefit-amount"
                value={values.amount}
                onChange={(value) => set("amount", value)}
                error={fields.amount_ars}
                disabled={pending}
                inputMode="numeric"
                maxLength={12}
                placeholder="2000"
                className="pl-9"
              />
            </div>
          </Field>
        ) : null}

        {values.type === BENEFIT_TYPE.CUSTOM ? (
          <Field
            id="benefit-title"
            label="Describí el beneficio"
            error={fields.title}
            hint={
              <span className="text-xs font-semibold text-movo-slate">
                {values.title.length}/{MAX_TITLE_LENGTH}
              </span>
            }
          >
            <TextInput
              id="benefit-title"
              value={values.title}
              onChange={(value) => set("title", value)}
              error={fields.title}
              disabled={pending}
              maxLength={MAX_TITLE_LENGTH}
              placeholder="Una clase de prueba gratis"
            />
          </Field>
        ) : null}

        <Field
          id="benefit-conditions"
          label="Condiciones"
          error={fields.conditions}
          hint={
            <span className="text-xs font-semibold text-movo-slate">
              Opcional · {values.conditions.length}/{MAX_CONDITIONS_LENGTH}
            </span>
          }
        >
          <TextArea
            id="benefit-conditions"
            value={values.conditions}
            onChange={(value) => set("conditions", value)}
            error={fields.conditions}
            disabled={pending}
            maxLength={MAX_CONDITIONS_LENGTH}
            rows={2}
          />
        </Field>

        <p className="rounded-2xl bg-movo-sand p-4 text-sm text-movo-slate">
          Vale en todas tus sedes. El cupón vence 48 hs después de generarlo. No incluyas links ni
          teléfonos.
        </p>

        <div className="flex items-center justify-between gap-4 rounded-2xl bg-movo-cream p-4">
          <div>
            <p className="font-semibold">
              {benefit ? "Beneficio activo" : "Activo inmediatamente"}
            </p>
            <p className="text-sm text-movo-slate">
              {benefit?.admin_paused
                ? "MOVO lo pausó: no se ofrece aunque esté activo."
                : "Visible desde ahora en la ficha del local."}
            </p>
          </div>
          <Switch
            checked={active}
            label="Beneficio activo"
            disabled={pending}
            onChange={setActive}
          />
        </div>

        {notice ? <Notice title={notice.title} body={notice.body} /> : null}

        <button
          type="submit"
          disabled={pending}
          className="flex h-14 w-full items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-60"
        >
          {pending ? "Guardando…" : benefit ? "Guardar cambios" : "Crear beneficio"}
          <CheckIcon />
        </button>
      </form>
    </dialog>
  );
}

function TypeButton({
  selected,
  disabled,
  onSelect,
  children,
}: {
  selected: boolean;
  disabled: boolean;
  onSelect: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      disabled={disabled}
      onClick={onSelect}
      className={`flex min-h-20 flex-col items-center justify-center gap-1 rounded-2xl px-1 text-center text-xs font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink ${
        selected ? "bg-movo-accent" : "bg-movo-sand"
      }`}
    >
      {children}
    </button>
  );
}
