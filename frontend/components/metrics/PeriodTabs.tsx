"use client";

import { periodLabel } from "@/lib/metrics/messages";
import { PERIODS, type Period } from "@/lib/metrics/types";

export function PeriodTabs({
  value,
  onChange,
}: {
  value: Period;
  onChange: (period: Period) => void;
}) {
  return (
    <div role="group" aria-label="Período" className="flex rounded-full bg-movo-sand p-1">
      {PERIODS.map((days) => (
        <button
          key={days}
          type="button"
          aria-pressed={value === days}
          onClick={() => onChange(days)}
          className={`h-10 flex-1 rounded-full text-sm font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent ${
            value === days ? "bg-movo-page text-white shadow-sm" : "text-movo-slate"
          }`}
        >
          {periodLabel(days)}
        </button>
      ))}
    </div>
  );
}
