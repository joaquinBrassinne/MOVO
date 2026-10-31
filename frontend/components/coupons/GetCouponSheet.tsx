"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type SubmitEvent } from "react";

import { Field, TextInput } from "@/components/onboarding/fields";
import { ArrowRightIcon, ClockIcon, InfoIcon, XIcon } from "@/components/socios/icons";
import { benefitHeadline } from "@/lib/benefits/format";
import type { ClaimedBenefit } from "@/lib/benefits/types";
import { toWhatsApp } from "@/lib/business/phone";
import { generateCoupon } from "@/lib/coupons/api";
import {
  describeGenerateFailure,
  GENERATE_PROBLEM,
  type GenerateFeedback,
} from "@/lib/coupons/messages";
import { saveToken } from "@/lib/coupons/saved";

interface Props {
  businessName: string;
  benefit: ClaimedBenefit;
  onClose: () => void;
}

/**
 * "Obtené tu cupón": el cliente deja nombre y WhatsApp, sin cuenta ni verificación, y recibe su
 * cupón en el acto. El enlace de acceso queda guardado en este dispositivo.
 */
export function GetCouponSheet({ businessName, benefit, onClose }: Props) {
  const router = useRouter();
  const dialog = useRef<HTMLDialogElement>(null);
  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [feedback, setFeedback] = useState<GenerateFeedback>({ fields: {} });
  const [pending, setPending] = useState(false);

  // <dialog> da el foco atrapado, el cierre con Escape y el fondo oscuro sin código extra.
  useEffect(() => {
    const element = dialog.current;
    if (element && !element.open) element.showModal();
  }, []);

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending) return;

    const whatsapp = toWhatsApp(phone);
    const fields: Record<string, string> = {};
    if (name.trim() === "") fields.name = "Ingresá tu nombre";
    if (whatsapp === null) fields.phone = "Ingresá tu número con característica, sin 0 ni 15";
    if (Object.keys(fields).length > 0 || whatsapp === null) {
      setFeedback({ fields });
      return;
    }

    setPending(true);
    setFeedback({ fields: {} });
    const result = await generateCoupon(benefit.id, { name: name.trim(), phone: whatsapp });
    if (!result.ok) {
      setPending(false);
      setFeedback(describeGenerateFailure(result.error, new Date()));
      return;
    }

    // El token solo viene en esta respuesta: se guarda antes de salir de la pantalla.
    saveToken(result.data.access_token);
    // Un diálogo modal abierto deja inerte toda la página aunque Next conserve la pantalla anterior
    // oculta: se cierra antes de navegar.
    dialog.current?.close();
    router.push(`/coupon/${result.data.access_token}?nuevo=1`);
  }

  const { fields, problem } = feedback;

  return (
    <dialog
      ref={dialog}
      onClose={onClose}
      aria-labelledby="get-coupon-title"
      className="fixed inset-x-0 bottom-0 mx-auto mt-auto max-h-[92dvh] w-full max-w-md overflow-y-auto rounded-t-[2rem] bg-movo-cream p-0 text-movo-ink backdrop:bg-black/50"
    >
      <form onSubmit={submit} noValidate aria-busy={pending} className="space-y-5 p-6 pb-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h2 id="get-coupon-title" className="font-display text-3xl font-extrabold">
              Obtené tu cupón
            </h2>
            <p className="mt-1 text-sm text-movo-slate">
              Completá tus datos y lo recibís en el acto.
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Cerrar"
            className="grid size-11 shrink-0 place-items-center rounded-full bg-white outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
          >
            <XIcon />
          </button>
        </div>

        <div className="space-y-2 rounded-3xl bg-white p-5 shadow-sm">
          <div className="flex items-center justify-between gap-3">
            <span className="rounded-full bg-movo-accent px-3 py-1 text-xs font-extrabold">
              {benefit.short_label}
            </span>
            <span className="inline-flex items-center gap-1.5 text-xs font-semibold text-movo-slate">
              <ClockIcon className="size-4" />
              Vence en 48 hs
            </span>
          </div>
          <p className="font-display text-xl leading-tight font-bold">{benefitHeadline(benefit)}</p>
          {benefit.conditions ? (
            <p className="text-sm text-movo-slate">{benefit.conditions}</p>
          ) : null}
          <p className="text-xs font-bold text-movo-slate">{businessName}</p>
        </div>

        <Field id="coupon-name" label="Tu nombre" error={fields.name}>
          <TextInput
            id="coupon-name"
            value={name}
            onChange={setName}
            error={fields.name}
            disabled={pending}
            autoComplete="name"
            maxLength={100}
            placeholder="Ej. Camila Gomez"
          />
        </Field>

        <Field id="coupon-phone" label="Tu WhatsApp" error={fields.phone}>
          <div className="flex gap-3">
            <span className="flex h-14 shrink-0 items-center rounded-2xl bg-movo-sand px-4 text-base font-semibold">
              AR +54
            </span>
            <TextInput
              id="coupon-phone"
              type="tel"
              inputMode="tel"
              autoComplete="tel-national"
              value={phone}
              onChange={setPhone}
              error={fields.phone}
              disabled={pending}
              placeholder="11 4567 8901"
            />
          </div>
        </Field>

        <p className="flex items-start gap-2 text-sm text-movo-slate">
          <InfoIcon className="mt-0.5 size-4 shrink-0 text-movo-gold" />
          Solo usamos tus datos para este cupón. No necesitás crear una cuenta ni contraseña.
        </p>

        {problem ? (
          <div
            role="alert"
            className={`space-y-2 rounded-2xl p-4 ${
              problem.kind === GENERATE_PROBLEM.PENDING ? "bg-movo-danger/10" : "bg-movo-sand"
            }`}
          >
            <p
              className={`font-display text-lg font-bold ${
                problem.kind === GENERATE_PROBLEM.PENDING ? "text-movo-danger" : ""
              }`}
            >
              {problem.title}
            </p>
            <p className="text-sm text-movo-slate">{problem.body}</p>
            {problem.kind === GENERATE_PROBLEM.PENDING ? (
              <Link
                href="/coupons"
                className="inline-flex h-11 items-center gap-2 rounded-full bg-white px-5 text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
              >
                Ver mi cupón activo
                <ArrowRightIcon className="size-4" />
              </Link>
            ) : null}
          </div>
        ) : null}

        <button
          type="submit"
          disabled={pending}
          className="flex h-14 w-full items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-60"
        >
          {pending ? "Generando…" : "Generar cupón"}
          {pending ? null : <ArrowRightIcon className="size-5" />}
        </button>
      </form>
    </dialog>
  );
}
