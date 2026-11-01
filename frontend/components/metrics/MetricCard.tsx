"use client";

import type { ReactNode } from "react";

import type { Loaded } from "@/lib/business/hooks";
import { failureText, insufficientText } from "@/lib/metrics/messages";

interface MetricCardProps<T extends object> {
  title: string;
  subtitle?: string;
  state: Loaded<T>;
  onRetry: () => void;
  /** Rendered with the data once it is ready and has enough sample. */
  children: (data: T) => ReactNode;
  /** The card shows its own empty state instead of the sample message (the quota, the ticket). */
  handlesStatus?: boolean;
  dark?: boolean;
  /** Right-hand badge of the header, such as "estimado". */
  badge?: string;
}

function Skeleton({ dark }: { dark: boolean }) {
  const tone = dark ? "bg-white/15" : "bg-movo-sand";
  return (
    <div role="status" aria-label="Cargando" className="mt-4 animate-pulse space-y-3">
      <div className={`h-8 w-1/2 rounded-xl ${tone}`} />
      <div className={`h-4 w-full rounded-xl ${tone}`} />
      <div className={`h-4 w-2/3 rounded-xl ${tone}`} />
    </div>
  );
}

/**
 * The shell of one metric: its title and the loading, failed and insufficient_data states, so a
 * card that fails never takes the page down with it.
 */
export function MetricCard<T extends object>({
  title,
  subtitle,
  state,
  onRetry,
  children,
  handlesStatus = false,
  dark = false,
  badge,
}: MetricCardProps<T>) {
  const muted = dark ? "text-movo-muted" : "text-movo-slate";

  let body: ReactNode;
  if (state.status === "loading") {
    body = <Skeleton dark={dark} />;
  } else if (state.status === "ready") {
    const data = state.data as { status?: string; minimum_sample?: number };
    body =
      !handlesStatus && data.status === "insufficient_data" ? (
        <p className={`mt-4 rounded-2xl bg-movo-sand/70 p-4 text-sm ${muted}`}>
          {insufficientText(data.minimum_sample)}
        </p>
      ) : (
        children(state.data)
      );
  } else {
    const code = state.status === "failed" ? state.error.code : "business_not_found";
    body = (
      <div role="alert" className="mt-4 space-y-3">
        <p className={`text-sm ${muted}`}>{failureText(code)}</p>
        {code !== "plan_upgrade_required" ? (
          <button
            type="button"
            onClick={onRetry}
            className="h-10 rounded-full bg-movo-sand px-5 text-sm font-semibold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
          >
            Reintentar
          </button>
        ) : null}
      </div>
    );
  }

  return (
    <section
      className={`rounded-3xl p-5 shadow-sm ${dark ? "bg-movo-page text-white" : "bg-white"}`}
    >
      <header className="flex items-start justify-between gap-3">
        <div>
          <h3 className="font-display text-lg font-extrabold">{title}</h3>
          {subtitle ? <p className={`text-sm ${muted}`}>{subtitle}</p> : null}
        </div>
        {badge ? (
          <span className="shrink-0 rounded-full bg-movo-accent px-3 py-1 text-xs font-bold text-movo-ink">
            {badge}
          </span>
        ) : null}
      </header>
      {body}
    </section>
  );
}

/** A small "+12 % vs período ant." chip; nothing when the API sent no baseline. */
export function ChangeChip({
  text,
  tone,
}: {
  text: string | null;
  tone: "up" | "down" | "flat" | null;
}) {
  if (!text || !tone) return null;
  const colour =
    tone === "up"
      ? "bg-movo-ok/15 text-movo-ok"
      : tone === "down"
        ? "bg-movo-danger/10 text-movo-danger"
        : "bg-movo-sand text-movo-slate";
  return (
    <span className={`rounded-full px-2.5 py-1 text-xs font-bold ${colour}`}>
      {text} vs período ant.
    </span>
  );
}

/** A horizontal bar for a fraction of 0 to 1. */
export function Bar({
  fraction,
  className = "bg-movo-accent",
}: {
  fraction: number;
  className?: string;
}) {
  const width = Math.max(0, Math.min(1, fraction)) * 100;
  return (
    <div className="h-2.5 w-full overflow-hidden rounded-full bg-movo-sand" aria-hidden="true">
      <div className={`h-full rounded-full ${className}`} style={{ width: `${width}%` }} />
    </div>
  );
}
