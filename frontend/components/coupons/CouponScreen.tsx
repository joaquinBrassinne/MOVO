"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { QrCode } from "@/components/benefits/QrCode";
import { CheckCircleIcon, ClockIcon, InfoIcon, XIcon } from "@/components/socios/icons";
import { benefitHeadline } from "@/lib/benefits/format";
import { displayCode } from "@/lib/benefits/code";
import { COUPON_STATUS } from "@/lib/benefits/types";
import { getHeldCoupon } from "@/lib/coupons/api";
import { formatCountdown, formatShortDate } from "@/lib/coupons/format";
import { isToken, readSaved, saveToken } from "@/lib/coupons/saved";
import type { HeldCoupon } from "@/lib/coupons/types";

/** Un instante fijo para antes de tener reloj: nunca se muestra, el cupón aparece junto con el reloj. */
const EPOCH = new Date(0);

type Loaded =
  | { status: "loading" }
  | { status: "ready"; coupon: HeldCoupon }
  | { status: "missing" }
  | { status: "failed" };

interface Props {
  token: string;
  /** Viene de generarlo recién: el título festeja en vez de solo mostrarlo. */
  fresh: boolean;
}

/**
 * "Cupón generado": el código y su QR (que lleva solo el código), lo que falta para que
 * venza y cómo guardarlo. Sirve para el cupón recién generado y para volver a abrirlo con su
 * enlace de acceso, que es la única forma de recuperarlo: no hay cuenta.
 */
export function CouponScreen({ token, fresh }: Props) {
  const [fetched, setFetched] = useState<Loaded>({ status: "loading" });
  // Un enlace que no es un token ni se consulta: es un cupón que no existe.
  const loaded: Loaded = isToken(token) ? fetched : { status: "missing" };
  // El reloj se toma al cargar y se mueve solo: leerlo al renderizar no es estable para Next.
  const [clock, setClock] = useState<Date | null>(null);
  const now = clock ?? EPOCH;
  const [notice, setNotice] = useState("");

  useEffect(() => {
    let ignore = false;
    if (!isToken(token)) return;

    void getHeldCoupon(token).then((result) => {
      if (ignore) return;
      if (result.ok) {
        setFetched({ status: "ready", coupon: result.data });
        setClock(new Date());
      } else setFetched(result.status === 404 ? { status: "missing" } : { status: "failed" });
    });
    return () => {
      ignore = true;
    };
  }, [token]);

  // La cuenta regresiva se mueve sola cada medio minuto: alcanza para horas y minutos.
  useEffect(() => {
    const timer = setInterval(() => setClock(new Date()), 30_000);
    return () => clearInterval(timer);
  }, []);

  async function copy(text: string, said: string) {
    try {
      await navigator.clipboard.writeText(text);
      setNotice(said);
    } catch {
      setNotice("No pudimos copiar. Seleccionalo a mano.");
    }
  }

  /** Guarda el cupón en este dispositivo y copia el enlace para volver a verlo desde otro. */
  async function keepLink() {
    const saved = readSaved().some((item) => item.token === token);
    saveToken(token);
    await copy(
      `${window.location.origin}/cupon/${token}`,
      saved ? "Enlace copiado." : "Cupón guardado en este dispositivo y enlace copiado.",
    );
  }

  return (
    <div className="min-h-dvh bg-movo-page font-sans text-white">
      <header className="mx-auto flex h-16 max-w-md items-center justify-between px-5">
        <Link
          href="/cupones"
          aria-label="Ir a mis cupones"
          className="grid size-11 place-items-center rounded-full outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
        >
          <XIcon />
        </Link>
        <p className="font-display text-lg font-extrabold">MOVO</p>
        <span className="size-11" aria-hidden="true" />
      </header>

      <main className="mx-auto max-w-md space-y-5 px-5 pb-10">
        {loaded.status === "loading" ? (
          <div
            aria-busy="true"
            aria-label="Cargando tu cupón"
            className="h-96 animate-pulse rounded-3xl bg-movo-card"
          />
        ) : null}

        {loaded.status === "missing" ? (
          <Problem
            title="No encontramos este cupón"
            body="El enlace puede estar incompleto. Si lo guardaste en este dispositivo, buscalo en tus cupones."
          />
        ) : null}
        {loaded.status === "failed" ? (
          <Problem
            title="No pudimos cargar tu cupón"
            body="Revisá tu conexión y probá de nuevo en unos minutos."
          />
        ) : null}

        {loaded.status === "ready" ? (
          <Ticket
            token={token}
            coupon={loaded.coupon}
            fresh={fresh}
            now={now}
            notice={notice}
            onCopyCode={() => void copy(loaded.coupon.code, "Código copiado.")}
            onKeepLink={() => void keepLink()}
          />
        ) : null}
      </main>
    </div>
  );
}

function Problem({ title, body }: { title: string; body: string }) {
  return (
    <section role="alert" className="space-y-3 rounded-3xl bg-movo-card p-6 text-center">
      <h1 className="font-display text-2xl font-extrabold">{title}</h1>
      <p className="text-movo-muted">{body}</p>
      <Link
        href="/cupones"
        className="inline-flex h-12 items-center rounded-full bg-movo-accent px-6 font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-white"
      >
        Ver mis cupones
      </Link>
    </section>
  );
}

interface TicketProps {
  token: string;
  coupon: HeldCoupon;
  fresh: boolean;
  now: Date;
  notice: string;
  onCopyCode: () => void;
  onKeepLink: () => void;
}

function Ticket({ token, coupon, fresh, now, notice, onCopyCode, onKeepLink }: TicketProps) {
  const { status, benefit } = coupon;
  const active = status === COUPON_STATUS.PENDING && new Date(coupon.expires_at) > now;
  const shown = displayCode(coupon.code);

  return (
    <>
      <div className="space-y-2 pt-2 text-center">
        <p
          className={`inline-flex items-center gap-2 rounded-full px-4 py-1.5 text-xs font-extrabold tracking-wide uppercase ${
            active ? "bg-movo-tabs text-movo-accent" : "bg-movo-tabs text-movo-muted"
          }`}
        >
          <CheckCircleIcon className="size-4" />
          {active ? "Pase activo" : status === COUPON_STATUS.REDEEMED ? "Canjeado" : "Vencido"}
        </p>
        <h1 className="font-display text-4xl font-extrabold">
          {!active ? "Tu cupón" : fresh ? "¡Cupón generado!" : "Tu cupón"}
        </h1>
        <p className="text-movo-muted">
          {active
            ? "Presentalo en recepción al ingresar."
            : status === COUPON_STATUS.REDEEMED
              ? `Se canjeó el ${formatShortDate(coupon.redeemed_at ?? coupon.expires_at)}.`
              : "Pasaron las 48 horas desde que se generó. Podés generar otro."}
        </p>
      </div>

      <section
        aria-label="Cupón"
        className={`space-y-4 rounded-3xl bg-movo-cream p-6 text-movo-ink ${active ? "" : "opacity-80"}`}
      >
        <div className="flex items-start justify-between gap-3">
          <p className="text-xs font-extrabold tracking-widest text-movo-slate uppercase">
            {coupon.business_name}
          </p>
          <span className="shrink-0 rounded-full bg-movo-accent px-3 py-1.5 font-display text-base font-extrabold">
            {benefit.short_label}
          </span>
        </div>
        <div>
          <p className="font-display text-2xl leading-tight font-extrabold">
            {benefitHeadline(benefit)}
          </p>
          {benefit.conditions ? (
            <p className="mt-1 text-sm text-movo-slate">{benefit.conditions}</p>
          ) : null}
        </div>

        <div className="border-t-2 border-dashed border-movo-slate/30 pt-4">
          {active ? (
            <QrCode
              code={coupon.code}
              label={`Código QR del cupón ${shown}`}
              className="mx-auto size-56 rounded-2xl"
            />
          ) : null}
          <p
            aria-label={`Código del cupón ${shown}`}
            className="mt-3 text-center font-mono text-3xl font-extrabold tracking-[0.2em]"
          >
            {shown}
          </p>
          {active ? (
            <p className="mx-auto mt-3 flex w-fit items-center gap-2 rounded-full bg-movo-accent px-4 py-1.5 text-sm font-bold">
              <ClockIcon className="size-4" />
              Vence en {formatCountdown(coupon.expires_at, now)} hs
            </p>
          ) : null}
          {active ? (
            <p className="mt-3 text-center text-sm font-semibold text-movo-slate">
              Mostralo en el local
            </p>
          ) : null}
        </div>
      </section>

      {active ? (
        <div className="space-y-3">
          <button
            type="button"
            onClick={onKeepLink}
            className="flex h-14 w-full items-center justify-center rounded-full bg-movo-accent font-display text-lg font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-white"
          >
            Guardar enlace del cupón
          </button>
          <button
            type="button"
            onClick={onCopyCode}
            className="flex h-12 w-full items-center justify-center rounded-full font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
          >
            Copiar código
          </button>
          <p role="status" className="min-h-5 text-center text-sm font-semibold text-movo-accent">
            {notice}
          </p>
        </div>
      ) : null}

      {status === COUPON_STATUS.REDEEMED ? (
        <Link
          href={`/cupon/${token}/review`}
          className="flex h-14 w-full items-center justify-center rounded-full bg-movo-accent font-display text-lg font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-white"
        >
          Dejá tu reseña
        </Link>
      ) : null}

      <p className="flex items-start gap-3 rounded-2xl bg-movo-tabs p-4 text-sm">
        <InfoIcon className="mt-0.5 size-5 shrink-0 text-movo-accent" />
        <span>
          <strong className="block">Sin registros ni contraseñas</strong>
          <span className="text-movo-muted">
            Guardá el enlace: es la forma de volver a ver tu cupón cuando llegues al local.
          </span>
        </span>
      </p>
    </>
  );
}
