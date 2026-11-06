"use client";

import { useState } from "react";

import { Switch } from "@/components/panel/Switch";
import { PlusIcon, XIcon } from "@/components/socios/icons";
import {
  addRange,
  copyDay,
  removeRange,
  setDayOpen,
  updateRange,
  type DayHours,
} from "@/lib/business/hours";
import { WEEK } from "@/lib/business/schedule";

interface Props {
  days: DayHours[];
  onChange: (days: DayHours[]) => void;
}

const TIME_INPUT =
  "h-11 min-w-0 flex-1 rounded-xl bg-movo-sand px-2 text-sm outline-none focus:ring-2 focus:ring-movo-accent";
const SMALL_BUTTON =
  "inline-flex h-9 items-center gap-1.5 rounded-full bg-movo-sand px-3 text-xs font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent";

const dayName = (weekday: number) => WEEK.find((d) => d.weekday === weekday)?.long ?? "";

/** Horarios de una sede por día, con turnos cortados y "Copiar a otros días". */
export function HoursEditor({ days, onChange }: Props) {
  // El día cuyo horario se está copiando y los días elegidos como destino.
  const [copying, setCopying] = useState<{ from: number; targets: number[] } | null>(null);

  function applyCopy() {
    if (copying) onChange(copyDay(days, copying.from, copying.targets));
    setCopying(null);
  }

  function toggleTarget(weekday: number) {
    setCopying((current) =>
      current
        ? {
            ...current,
            targets: current.targets.includes(weekday)
              ? current.targets.filter((day) => day !== weekday)
              : [...current.targets, weekday],
          }
        : current,
    );
  }

  return (
    <ul className="space-y-4">
      {days.map((row) => {
        const name = dayName(row.weekday);
        const open = row.ranges.length > 0;

        return (
          <li key={row.weekday} className="space-y-2">
            <div className="flex items-center gap-3">
              <Switch
                checked={open}
                label={`${name}: ${open ? "abierto" : "cerrado"}`}
                onChange={(next) => onChange(setDayOpen(days, row.weekday, next))}
              />
              <span className="w-24 shrink-0 text-sm font-semibold">{name}</span>
              {open ? null : (
                <span className="rounded-full bg-movo-sand px-3 py-1 text-xs font-bold text-movo-slate">
                  Cerrado
                </span>
              )}
            </div>

            {open ? (
              <div className="space-y-2 pl-1">
                {row.ranges.map((range, index) => {
                  const label = index === 0 ? name : `${name}, franja ${index + 1}`;
                  return (
                    <div key={index} className="flex items-center gap-2">
                      <input
                        type="time"
                        aria-label={`${label}: abre`}
                        value={range.start}
                        onChange={(event) =>
                          onChange(
                            updateRange(days, row.weekday, index, { start: event.target.value }),
                          )
                        }
                        className={TIME_INPUT}
                      />
                      <span className="text-sm text-movo-slate">a</span>
                      <input
                        type="time"
                        aria-label={`${label}: cierra`}
                        value={range.end}
                        onChange={(event) =>
                          onChange(
                            updateRange(days, row.weekday, index, { end: event.target.value }),
                          )
                        }
                        className={TIME_INPUT}
                      />
                      <button
                        type="button"
                        aria-label={`Quitar la franja ${index + 1} de ${name}`}
                        onClick={() => onChange(removeRange(days, row.weekday, index))}
                        className="grid size-9 shrink-0 place-items-center rounded-full bg-movo-sand outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
                      >
                        <XIcon className="size-4" />
                      </button>
                    </div>
                  );
                })}

                <div className="flex flex-wrap gap-2">
                  <button
                    type="button"
                    onClick={() => onChange(addRange(days, row.weekday))}
                    className={SMALL_BUTTON}
                  >
                    <PlusIcon className="size-4" />
                    Agregar franja
                  </button>
                  <button
                    type="button"
                    aria-label={`Copiar ${name} a otros días`}
                    onClick={() => setCopying({ from: row.weekday, targets: [] })}
                    className={SMALL_BUTTON}
                  >
                    Copiar a otros días
                  </button>
                </div>
              </div>
            ) : null}

            {copying?.from === row.weekday ? (
              <fieldset className="space-y-3 rounded-2xl bg-movo-sand p-4">
                <legend className="px-1 text-sm font-semibold">
                  Copiar el horario de {name} a
                </legend>
                <div className="flex flex-wrap gap-2">
                  {WEEK.filter((d) => d.weekday !== row.weekday).map((d) => (
                    <label
                      key={d.weekday}
                      className="inline-flex h-9 items-center gap-2 rounded-full bg-white px-3 text-sm font-semibold"
                    >
                      <input
                        type="checkbox"
                        checked={copying.targets.includes(d.weekday)}
                        onChange={() => toggleTarget(d.weekday)}
                        className="size-4 accent-movo-ink"
                      />
                      {d.long}
                    </label>
                  ))}
                </div>
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={applyCopy}
                    disabled={copying.targets.length === 0}
                    className="h-10 rounded-full bg-movo-accent px-5 text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-50"
                  >
                    Aplicar
                  </button>
                  <button
                    type="button"
                    onClick={() => setCopying(null)}
                    className="h-10 rounded-full px-4 text-sm font-semibold underline underline-offset-4"
                  >
                    Cancelar
                  </button>
                </div>
              </fieldset>
            ) : null}
          </li>
        );
      })}
    </ul>
  );
}
