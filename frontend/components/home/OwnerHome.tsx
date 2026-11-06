"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState, type ReactNode } from "react";

import { BellIcon, QrIcon } from "@/components/benefits/icons";
import { QuotaCard } from "@/components/benefits/QuotaCard";
import { Notice } from "@/components/onboarding/fields";
import { useActivity, useCouponQuota, useOverview } from "@/lib/benefits/hooks";
import { readSeenAt, writeSeenAt } from "@/lib/benefits/seen";
import { useBusiness } from "@/lib/business/BusinessProvider";

import { ActivityList } from "./ActivityList";

/** El inicio del panel: canjear, los números del local, el cupo del mes y la actividad. */
export function OwnerHome() {
  const router = useRouter();
  const { business } = useBusiness();
  const overview = useOverview();
  const quota = useCouponQuota();
  // La marca de lo ya visto: lo posterior a ella cuenta como novedad en la campana.
  // La primera vez no hay marca: se guarda el momento de la visita (abajo), así la próxima vez
  // cuenta lo que haya pasado desde entonces. Leer la hora al renderizar no es estable para Next.
  const [seenAt, setSeenAt] = useState<string | undefined>(() => readSeenAt());
  const activity = useActivity(seenAt);
  const activityRef = useRef<HTMLElement>(null);

  useEffect(() => {
    if (!readSeenAt()) writeSeenAt(new Date().toISOString());
  }, []);

  const name = business.status === "ready" ? business.data.business.name : "";
  const pilot = business.status === "ready" && business.data.business.pilot;
  const news = activity.state.status === "ready" ? activity.state.data.new_count : 0;
  const items = activity.state.status === "ready" ? activity.state.data.items : [];

  /** Marca todo como visto: la próxima consulta ya no cuenta lo que hay ahora. */
  function markSeen() {
    activityRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
    const newest = items[0]?.at;
    if (!newest) return;
    writeSeenAt(newest);
    setSeenAt(newest);
  }

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <h2 className="font-display text-3xl leading-tight font-extrabold">
            Hola{name ? `, ${name}` : ""}
          </h2>
          {pilot ? (
            <span className="mt-1 inline-block rounded-full bg-movo-accent px-3 py-1 text-xs font-extrabold tracking-wide uppercase">
              Local piloto
            </span>
          ) : null}
        </div>
        <button
          type="button"
          onClick={markSeen}
          aria-label={news > 0 ? `Novedades: ${news}` : "Novedades"}
          className="relative grid size-12 shrink-0 place-items-center rounded-full bg-white shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
        >
          <BellIcon className="size-6" />
          {news > 0 ? (
            <span className="absolute -top-1 -right-1 grid min-w-5 place-items-center rounded-full bg-movo-accent px-1 text-xs font-extrabold">
              {news > 99 ? "99+" : news}
            </span>
          ) : null}
        </button>
      </div>

      <div>
        <Link
          href="/owners/redeem"
          className="flex h-16 w-full items-center justify-center gap-3 rounded-full bg-movo-accent font-display text-xl font-extrabold shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
        >
          <QrIcon className="size-6" />
          Canjear cupón
        </Link>
        <p className="mt-2 text-center text-sm text-movo-slate">
          Escaneá el QR de tu cliente en segundos.
        </p>
      </div>

      {overview.state.status === "ready" ? (
        <div className="grid grid-cols-2 gap-3">
          <Stat
            label="Cupones activos"
            value={overview.state.data.active_coupons}
            note="Disponibles"
          />
          <Stat label="Canjes de hoy" value={overview.state.data.redeemed_today} note="Hoy" />
          <Stat
            label="Vistas de tu perfil"
            value={overview.state.data.profile_views_7d}
            note="Últimos 7 días"
          />
          <Stat
            label="Clics a WhatsApp"
            value={overview.state.data.whatsapp_clicks_7d}
            note="Últimos 7 días"
          />
        </div>
      ) : overview.state.status === "loading" ? (
        <div aria-busy="true" aria-label="Cargando tus números" className="grid grid-cols-2 gap-3">
          {[0, 1, 2, 3].map((key) => (
            <div key={key} className="h-32 animate-pulse rounded-3xl bg-white" />
          ))}
        </div>
      ) : (
        <Notice title="No pudimos cargar tus números" body="Probá de nuevo en unos minutos." />
      )}

      {quota.state.status === "ready" ? (
        <QuotaCard quota={quota.state.data} onEdit={() => router.push("/owners/benefits")} />
      ) : null}

      <section
        ref={activityRef}
        aria-labelledby="activity-title"
        className="scroll-mt-24 rounded-3xl bg-white p-5 shadow-sm"
      >
        <h3 id="activity-title" className="font-display text-xl font-bold">
          Actividad reciente
        </h3>

        {activity.state.status === "loading" ? (
          <div aria-busy="true" aria-label="Cargando la actividad" className="mt-4 space-y-3">
            {[0, 1, 2].map((key) => (
              <div key={key} className="h-12 animate-pulse rounded-2xl bg-movo-sand" />
            ))}
          </div>
        ) : null}
        {activity.state.status === "failed" ? (
          <p className="mt-3 text-movo-slate">No pudimos cargar la actividad. Probá de nuevo.</p>
        ) : null}
        {activity.state.status === "ready" && items.length === 0 ? (
          <p className="mt-3 text-movo-slate">
            Todavía no hay actividad. Cuando un cliente genere un cupón, lo vas a ver acá.
          </p>
        ) : null}
        {items.length > 0 ? <ActivityList items={items} now={new Date()} /> : null}
      </section>
    </div>
  );
}

function Stat({ label, value, note }: { label: string; value: number; note: ReactNode }) {
  return (
    <div className="rounded-3xl bg-white p-4 shadow-sm">
      <p className="text-sm font-semibold">{label}</p>
      <p className="mt-2 font-display text-4xl font-extrabold">{value}</p>
      <p className="mt-1 text-xs font-semibold text-movo-slate">{note}</p>
    </div>
  );
}
