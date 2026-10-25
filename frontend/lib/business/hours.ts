// Horarios de atención de una sede: por día, cero o más franjas (turno cortado). La API
// guarda tramos sueltos; acá se pasa de un lado al otro y se editan las franjas.
// Un día cerrado es un día sin franjas. Ninguna franja cruza la medianoche.

import { findOverlap, isValidRange, WEEK, type Range } from "./schedule.ts";
import type { Slot } from "./types.ts";

export interface DayHours {
  weekday: number;
  ranges: Range[];
}

/** La franja que se propone al abrir un día que estaba cerrado. */
const DEFAULT_RANGE: Range = { start: "09:00", end: "18:00" };

/** Los siete días en el orden de la semana, con las franjas de cada uno ordenadas. */
export function toDayHours(slots: Slot[]): DayHours[] {
  return WEEK.map(({ weekday }) => ({
    weekday,
    ranges: slots
      .filter((slot) => slot.weekday === weekday)
      .map(({ start, end }) => ({ start, end }))
      .sort((a, b) => a.start.localeCompare(b.start)),
  }));
}

/** Los tramos que se guardan: las franjas de cada día, en el orden de la semana. */
export function toSlots(rows: DayHours[]): Slot[] {
  return rows.flatMap(({ weekday, ranges }) =>
    ranges.map(({ start, end }) => ({ weekday, start, end })),
  );
}

function changeDay(rows: DayHours[], weekday: number, ranges: (current: Range[]) => Range[]) {
  return rows.map((row) =>
    row.weekday === weekday ? { ...row, ranges: ranges(row.ranges) } : row,
  );
}

/** Abrir un día le pone la franja por defecto; cerrarlo le quita todas. */
export function setDayOpen(rows: DayHours[], weekday: number, open: boolean): DayHours[] {
  return changeDay(rows, weekday, () => (open ? [{ ...DEFAULT_RANGE }] : []));
}

/**
 * "Agregar franja": arranca donde termina la última (una hora de largo) para que no se
 * pise; si no entra más día, repite la franja por defecto y el dueño la corrige.
 */
export function addRange(rows: DayHours[], weekday: number): DayHours[] {
  return changeDay(rows, weekday, (current) => {
    const last = current.at(-1);
    if (!last) return [{ ...DEFAULT_RANGE }];
    const hour = Number(last.end.slice(0, 2));
    if (!isValidRange(last) || hour >= 22) return [...current, { ...DEFAULT_RANGE }];
    const start = last.end;
    const end = `${String(hour + 1).padStart(2, "0")}${last.end.slice(2)}`;
    return [...current, { start, end }];
  });
}

export function removeRange(rows: DayHours[], weekday: number, index: number): DayHours[] {
  return changeDay(rows, weekday, (current) => current.filter((_, i) => i !== index));
}

export function updateRange(
  rows: DayHours[],
  weekday: number,
  index: number,
  change: Partial<Range>,
): DayHours[] {
  return changeDay(rows, weekday, (current) =>
    current.map((range, i) => (i === index ? { ...range, ...change } : range)),
  );
}

/** "Copiar a otros días": las franjas de `from` reemplazan las de cada día de `targets`. */
export function copyDay(rows: DayHours[], from: number, targets: number[]): DayHours[] {
  const source = rows.find((row) => row.weekday === from);
  if (!source) return rows;

  return rows.map((row) =>
    row.weekday !== from && targets.includes(row.weekday)
      ? { ...row, ranges: source.ranges.map((range) => ({ ...range })) }
      : row,
  );
}

export interface HoursProblem {
  weekday: number;
  kind: "invalid" | "overlap";
}

/** El primer día con una franja mal armada o dos franjas que se pisan, o `null` si todo está bien. */
export function firstProblem(rows: DayHours[]): HoursProblem | null {
  for (const { weekday, ranges } of rows) {
    if (!ranges.every(isValidRange)) return { weekday, kind: "invalid" };
    if (findOverlap(ranges)) return { weekday, kind: "overlap" };
  }
  return null;
}
