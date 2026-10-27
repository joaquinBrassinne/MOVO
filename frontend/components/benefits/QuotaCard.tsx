import { formatResetDate, QUOTA_LEVEL, summarizeQuota } from "@/lib/benefits/quota";
import type { CouponQuota } from "@/lib/benefits/types";

import { AlertTriangleIcon, CalendarClockIcon, TicketIcon } from "./icons";

interface Props {
  quota: CouponQuota;
  onEdit: () => void;
}

/** "Cupo del mes": cuántos cupones generó el local y cuántos le quedan, con la barra. */
export function QuotaCard({ quota, onEdit }: Props) {
  const summary = summarizeQuota(quota);
  const resets = formatResetDate(quota.resets_on);
  const unlimited = summary.level === QUOTA_LEVEL.UNLIMITED;
  const low = summary.level === QUOTA_LEVEL.LOW;
  const exhausted = summary.level === QUOTA_LEVEL.EXHAUSTED;

  return (
    <section
      aria-label="Cupo del mes"
      className="space-y-4 rounded-3xl bg-white p-5 shadow-sm"
      data-level={summary.level}
    >
      <div className="flex items-center justify-between gap-3">
        <h3 className="flex items-center gap-2 font-display text-lg font-bold">
          <span className="grid size-9 place-items-center rounded-full bg-movo-sand">
            <TicketIcon className="size-5" />
          </span>
          Cupo del mes
        </h3>
        <button
          type="button"
          onClick={onEdit}
          className="h-10 rounded-full bg-movo-sand px-4 text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
        >
          Editar cupo
        </button>
      </div>

      {unlimited ? (
        <div>
          <p className="font-display text-4xl font-extrabold">Sin límite</p>
          <p className="mt-1 text-sm text-movo-slate">
            Este mes se {summary.used === 1 ? "generó" : "generaron"} {summary.used}{" "}
            {summary.used === 1 ? "cupón" : "cupones"}. Podés ponerle un tope al mes.
          </p>
        </div>
      ) : (
        <>
          <div className="flex items-end justify-between gap-3">
            <p className="font-display text-4xl font-extrabold">
              {summary.used}
              <span className="text-xl font-bold text-movo-slate"> de {summary.limit}</span>
            </p>
            {low ? (
              <span className="rounded-full bg-movo-accent px-3 py-1 text-xs font-bold">
                Quedan pocos
              </span>
            ) : null}
          </div>
          <p className="-mt-2 text-sm text-movo-slate">cupones generados este mes</p>

          <div
            role="progressbar"
            aria-label="Cupo consumido"
            aria-valuemin={0}
            aria-valuemax={100}
            aria-valuenow={summary.percent}
            className="h-3 overflow-hidden rounded-full bg-movo-sand"
          >
            <div
              className={`h-full rounded-full ${exhausted ? "bg-movo-danger" : "bg-movo-accent"}`}
              style={{ width: `${summary.percent}%` }}
            />
          </div>
          <div className="flex justify-between text-sm font-semibold">
            <span>{summary.percent}% consumido</span>
            <span>
              {summary.remaining} {summary.remaining === 1 ? "restante" : "restantes"}
            </span>
          </div>
        </>
      )}

      {exhausted ? (
        <p
          role="status"
          className="flex items-start gap-3 rounded-2xl bg-movo-danger/10 p-4 text-sm font-semibold"
        >
          <AlertTriangleIcon className="mt-0.5 size-5 shrink-0 text-movo-danger" />
          <span>
            Llegaste al cupo de este mes: tus beneficios no entregan más cupones
            {resets ? ` hasta el ${resets}` : ""}. Podés ampliar el cupo cuando quieras.
          </span>
        </p>
      ) : null}

      {resets ? (
        <p className="flex items-center gap-2 text-sm text-movo-slate">
          <CalendarClockIcon className="size-4 shrink-0" />
          Se reinicia el {resets}
        </p>
      ) : null}
    </section>
  );
}
