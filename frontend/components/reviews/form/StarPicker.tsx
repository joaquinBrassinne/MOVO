"use client";

import { useRef } from "react";
import type { KeyboardEvent } from "react";

import { cn } from "@/lib/cn";

const STAR_PATH =
  "M12 2.8a.6.6 0 0 1 1 0l2.6 5.6 6.1.8a.6.6 0 0 1 .3 1l-4.5 4.3 1.1 6a.6.6 0 0 1-.9.7L12 18.3l-5.4 2.9a.6.6 0 0 1-.9-.7l1.1-6L2.3 10.2a.6.6 0 0 1 .3-1l6.1-.8Z";

const STARS = [1, 2, 3, 4, 5];

/**
 * Interactive 1-5 star picker as a radio group: arrows move and pick, Tab enters the
 * checked star (or the first one when none is picked).
 */
export function StarPicker({
  value,
  onChange,
  invalid = false,
  describedBy,
}: {
  value: number;
  onChange: (value: number) => void;
  invalid?: boolean;
  describedBy?: string;
}) {
  const refs = useRef<(HTMLButtonElement | null)[]>([]);

  function pick(next: number) {
    onChange(next);
    refs.current[next - 1]?.focus();
  }

  function onKeyDown(event: KeyboardEvent, star: number) {
    if (event.key === "ArrowRight" || event.key === "ArrowUp") {
      event.preventDefault();
      pick(Math.min(5, star + 1));
    } else if (event.key === "ArrowLeft" || event.key === "ArrowDown") {
      event.preventDefault();
      pick(Math.max(1, star - 1));
    }
  }

  return (
    <div
      role="radiogroup"
      aria-label="Tu puntuación"
      aria-invalid={invalid || undefined}
      aria-describedby={describedBy}
      className="flex justify-center gap-2"
    >
      {STARS.map((star) => (
        <button
          key={star}
          ref={(node) => {
            refs.current[star - 1] = node;
          }}
          type="button"
          role="radio"
          aria-checked={value === star}
          aria-label={`${star} ${star === 1 ? "estrella" : "estrellas"}`}
          tabIndex={value === star || (value === 0 && star === 1) ? 0 : -1}
          onClick={() => onChange(star)}
          onKeyDown={(event) => onKeyDown(event, star)}
          className="grid size-11 place-items-center rounded-full outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
        >
          <svg
            viewBox="0 0 24 24"
            aria-hidden="true"
            className={cn(
              "size-9",
              star <= value
                ? "fill-movo-accent stroke-movo-accent"
                : "fill-none stroke-movo-slate/60",
            )}
            strokeWidth={1.4}
            strokeLinejoin="round"
          >
            <path d={STAR_PATH} />
          </svg>
        </button>
      ))}
    </div>
  );
}
