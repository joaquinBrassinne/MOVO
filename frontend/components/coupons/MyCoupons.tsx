"use client";

import Link from "next/link";
import { useEffect, useState, useSyncExternalStore } from "react";

import { QrIcon } from "@/components/benefits/icons";
import { AccountLink, AppFrame } from "@/components/explorar/AppFrame";
import { Logo } from "@/components/Logo";
import { CheckCircleIcon, ClockIcon, InfoIcon } from "@/components/socios/icons";
import { displayCode } from "@/lib/benefits/code";
import { benefitHeadline } from "@/lib/benefits/format";
import { COUPON_STATUS } from "@/lib/benefits/types";
import { cn } from "@/lib/cn";
import { getHeldCoupon } from "@/lib/coupons/api";
import { formatExpiresIn, formatShortDate } from "@/lib/coupons/format";
import { parseSaved, readRawSaved, subscribeSaved } from "@/lib/coupons/saved";
import type { HeldCoupon } from "@/lib/coupons/types";

interface Item {
  token: string;
  coupon: HeldCoupon;
}

const TABS = [
  { key: "all", label: "Todos" },
  { key: "active", label: "Activos" },
  { key: "history", label: "Historial" },
] as const;

type TabKey = (typeof TABS)[number]["key"];

const isActive = (coupon: HeldCoupon, now: Date) =>
  coupon.status === COUPON_STATUS.PENDING && new Date(coupon.expires_at) > now;

/** Mis cupones: los que el cliente guardó en este dispositivo, cada uno con el enlace para abrirlo. */
export function MyCoupons() {
  // Lo guardado vive en el navegador: en el servidor no hay nada y recién al montar se lee.
  const raw = useSyncExternalStore(subscribeSaved, readRawSaved, () => null);
  const tokens = parseSaved(raw)
    .map((item) => item.token)
    .join(",");

  const [items, setItems] = useState<Item[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [tab, setTab] = useState<TabKey>("all");
  const [now] = useState(() => new Date());

  useEffect(() => {
    let ignore = false;
    const list = tokens === "" ? [] : tokens.split(",");

    void Promise.all(
      list.map((token) => getHeldCoupon(token).then((result) => ({ token, result }))),
    ).then((answers) => {
      if (ignore) return;
      const found: Item[] = [];
      let broken = false;
      for (const { token, result } of answers) {
        if (result.ok) found.push({ token, coupon: result.data });
        // Un token que el servidor no conoce no se muestra; una falla de red sí se avisa.
        else if (result.status !== 404) broken = true;
      }
      setItems(found);
      setFailed(broken);
    });
    return () => {
      ignore = true;
    };
  }, [tokens]);

  const loaded = tokens === "" ? [] : items;
  const all = loaded ?? [];
  const active = all.filter((item) => isActive(item.coupon, now));
  const history = all.filter((item) => !isActive(item.coupon, now));
  const shown = tab === "active" ? active : tab === "history" ? history : all;
  const counts: Record<TabKey, number> = {
    all: all.length,
    active: active.length,
    history: history.length,
  };

  return (
    <AppFrame>
      <header className="flex items-center justify-between px-5 pt-5 pb-4">
        <Logo className="h-9" />
        <AccountLink />
      </header>

      <div className="min-h-[70dvh] space-y-5 rounded-t-[2rem] bg-movo-cream px-4 pt-5 pb-8 text-movo-ink">
        <p className="flex items-start gap-3 rounded-2xl bg-white p-4 text-sm shadow-sm">
          <InfoIcon className="mt-0.5 size-5 shrink-0 text-movo-gold" />
          <span>
            <strong className="block">Tus cupones se guardan en este dispositivo</strong>
            <span className="text-movo-slate">
              Sin registros ni contraseñas. Quedan listos en tu navegador para mostrar al ingresar.
            </span>
          </span>
        </p>

        <div className="flex items-baseline justify-between">
          <h1 className="font-display text-3xl font-extrabold">Mis cupones</h1>
          <span className="text-sm font-semibold text-movo-slate">
            {active.length} {active.length === 1 ? "activo" : "activos"}
          </span>
        </div>

        <div role="tablist" aria-label="Filtrar cupones" className="flex gap-2">
          {TABS.map(({ key, label }) => (
            <button
              key={key}
              type="button"
              role="tab"
              aria-selected={tab === key}
              onClick={() => setTab(key)}
              className={cn(
                "h-11 rounded-full px-4 text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent",
                tab === key ? "bg-movo-card text-white" : "bg-white",
              )}
            >
              {label} <span className="font-semibold opacity-70">({counts[key]})</span>
            </button>
          ))}
        </div>

        {loaded === null ? (
          <div aria-busy="true" aria-label="Cargando tus cupones" className="space-y-3">
            {[0, 1].map((key) => (
              <div key={key} className="h-40 animate-pulse rounded-3xl bg-white" />
            ))}
          </div>
        ) : null}

        {failed ? (
          <p role="alert" className="rounded-2xl bg-movo-danger/10 p-4 text-sm">
            No pudimos cargar algunos cupones. Revisá tu conexión y volvé a abrir esta pantalla.
          </p>
        ) : null}

        {loaded !== null && loaded.length === 0 && !failed ? (
          <section className="space-y-3 rounded-3xl bg-white p-6 text-center shadow-sm">
            <h2 className="font-display text-xl font-bold">Todavía no tenés cupones guardados</h2>
            <p className="text-movo-slate">
              Elegí un local con beneficios y generá tu cupón: queda acá, en este dispositivo.
            </p>
            <Link
              href="/mapa"
              className="inline-flex h-12 items-center rounded-full bg-movo-accent px-6 font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
            >
              Explorar locales
            </Link>
          </section>
        ) : null}

        {loaded !== null && loaded.length > 0 && shown.length === 0 ? (
          <p className="rounded-3xl bg-white p-6 text-center text-movo-slate shadow-sm">
            No hay cupones en esta lista.
          </p>
        ) : null}

        <ul aria-label="Cupones" className="space-y-3">
          {shown.map((item) => (
            <CouponCard key={item.token} item={item} now={now} />
          ))}
        </ul>

        <section className="space-y-2 rounded-3xl bg-white p-5 shadow-sm">
          <h2 className="font-display text-lg font-bold">¿Cómo presento mi cupón?</h2>
          <p className="text-sm text-movo-slate">
            Llegás al local, abrís el cupón con su código y el QR y se lo mostrás al personal. No
            necesitás imprimir nada ni dar tus datos personales.
          </p>
        </section>
      </div>
    </AppFrame>
  );
}

function CouponCard({ item, now }: { item: Item; now: Date }) {
  const { coupon, token } = item;
  const { benefit } = coupon;
  const live = isActive(coupon, now);
  const redeemed = coupon.status === COUPON_STATUS.REDEEMED;
  const code = displayCode(coupon.code);

  return (
    <li className={cn("space-y-3 rounded-3xl bg-white p-5 shadow-sm", !live && "opacity-70")}>
      <div className="flex items-start justify-between gap-3">
        <p className="min-w-0 truncate text-sm font-bold text-movo-slate">{coupon.business_name}</p>
        <span
          className={cn(
            "shrink-0 rounded-full px-3 py-1 text-xs font-extrabold",
            live ? "bg-movo-accent" : "bg-movo-sand text-movo-slate",
          )}
        >
          {benefit.short_label}
        </span>
      </div>
      <h3 className="font-display text-xl leading-tight font-extrabold">
        {benefitHeadline(benefit)}
      </h3>
      {benefit.conditions ? <p className="text-sm text-movo-slate">{benefit.conditions}</p> : null}

      <div className="flex items-center justify-between gap-3 rounded-2xl bg-movo-cream px-4 py-3">
        <span className="flex items-center gap-2 font-mono text-base font-extrabold tracking-widest">
          <QrIcon className="size-5" />
          {code}
        </span>
        {live ? (
          <span className="flex items-center gap-1.5 rounded-full bg-movo-accent px-3 py-1 text-xs font-bold">
            <ClockIcon className="size-3.5" />
            {formatExpiresIn(coupon.expires_at, now)}
          </span>
        ) : redeemed ? (
          <span className="flex items-center gap-1.5 text-xs font-bold text-movo-ok">
            <CheckCircleIcon className="size-4" />
            Canjeado el {formatShortDate(coupon.redeemed_at ?? coupon.expires_at)}
          </span>
        ) : (
          <span className="text-xs font-bold text-movo-slate">
            Venció el {formatShortDate(coupon.expires_at)}
          </span>
        )}
      </div>

      {live ? (
        <div className="flex items-center justify-between gap-3 text-sm font-bold">
          <span className="text-movo-gold">Mostralo en recepción</span>
          <Link
            href={`/cupon/${token}`}
            className="outline-none focus-visible:underline"
            aria-label={`Abrir el cupón ${code} y su QR`}
          >
            Abrir cupón y QR →
          </Link>
        </div>
      ) : (
        <p className="text-right text-xs text-movo-slate">
          {redeemed ? "Cupón ya utilizado" : "Cupón vencido"}
        </p>
      )}
    </li>
  );
}
