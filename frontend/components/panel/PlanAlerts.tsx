"use client";

import Link from "next/link";
import { useState } from "react";

import { AlertCircleIcon } from "@/components/socios/icons";
import { alertView } from "@/lib/subscription/messages";
import type { PlanAlert } from "@/lib/subscription/types";

interface Props {
  alerts: PlanAlert[];
  /** Marks the alert read; resolves false when it could not. */
  onDismiss: (id: number) => Promise<boolean>;
}

/** One warm notice per alert: a plan change or coupons that are not being redeemed. */
export function PlanAlerts({ alerts, onDismiss }: Props) {
  const [busy, setBusy] = useState<number | null>(null);
  const [failed, setFailed] = useState<number | null>(null);

  async function dismiss(id: number) {
    setBusy(id);
    setFailed(null);
    const ok = await onDismiss(id);
    setBusy(null);
    if (!ok) setFailed(id);
  }

  // An alert of a type this build does not know is skipped, never rendered half-empty.
  const items = alerts.flatMap((alert) => {
    const view = alertView(alert);
    return view ? [{ alert, view }] : [];
  });
  if (items.length === 0) return null;

  return (
    <ul className="space-y-3">
      {items.map(({ alert, view }) => (
        <li
          key={alert.id}
          className="flex gap-3 rounded-3xl border-l-4 border-movo-accent bg-white p-4 shadow-sm"
        >
          <AlertCircleIcon className="mt-0.5 size-5 shrink-0 text-movo-ink" />
          <div className="min-w-0 flex-1">
            <h3 className="font-display text-base font-bold">{view.title}</h3>
            <p className="mt-1 text-sm text-movo-slate">{view.body}</p>
            {failed === alert.id ? (
              <p role="alert" className="mt-2 text-sm font-semibold text-movo-danger">
                No pudimos marcarlo como leído. Probá de nuevo.
              </p>
            ) : null}
            <div className="mt-3 flex flex-wrap items-center gap-3">
              {view.href ? (
                <Link
                  href={view.href}
                  className="flex h-10 items-center rounded-full bg-movo-accent px-5 text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
                >
                  {view.linkLabel}
                </Link>
              ) : null}
              <button
                type="button"
                disabled={busy === alert.id}
                onClick={() => void dismiss(alert.id)}
                className="h-10 rounded-full bg-movo-sand px-5 text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-60"
              >
                Entendido
              </button>
            </div>
          </div>
        </li>
      ))}
    </ul>
  );
}
