"use client";

import { useState } from "react";

import { AlertCircleIcon } from "@/components/socios/icons";
import { downgradeMessage } from "@/lib/subscription/messages";
import type { PlanAlert } from "@/lib/subscription/types";

interface Props {
  alerts: PlanAlert[];
  /** Marks the alert read; resolves false when it could not. */
  onDismiss: (id: number) => Promise<boolean>;
}

/** One warm notice per plan change that switched branches or services off. */
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

  if (alerts.length === 0) return null;

  return (
    <ul className="space-y-3">
      {alerts.map((alert) => (
        <li
          key={alert.id}
          className="flex gap-3 rounded-3xl border-l-4 border-movo-accent bg-white p-4 shadow-sm"
        >
          <AlertCircleIcon className="mt-0.5 size-5 shrink-0 text-movo-ink" />
          <div className="min-w-0 flex-1">
            <h3 className="font-display text-base font-bold">Cambió tu plan</h3>
            <p className="mt-1 text-sm text-movo-slate">{downgradeMessage(alert.payload)}</p>
            {failed === alert.id ? (
              <p role="alert" className="mt-2 text-sm font-semibold text-movo-danger">
                No pudimos marcarlo como leído. Probá de nuevo.
              </p>
            ) : null}
            <button
              type="button"
              disabled={busy === alert.id}
              onClick={() => void dismiss(alert.id)}
              className="mt-3 h-10 rounded-full bg-movo-sand px-5 text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-60"
            >
              Entendido
            </button>
          </div>
        </li>
      ))}
    </ul>
  );
}
