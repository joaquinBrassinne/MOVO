// Horarios semanales. La API guarda tramos sueltos (día, desde, hasta); el dueño los
// carga como "estos días, estas franjas". Acá se convierte de una forma a la otra.

import type { Slot } from "./types.ts";

export interface Range {
  start: string;
  end: string;
}

export interface Weekly {
  days: number[];
  ranges: Range[];
}

/** Los días como los muestra la interfaz (la semana empieza el lunes) y como los guarda la API. */
export const WEEK = [
  { weekday: 1, short: "Lun", long: "Lunes" },
  { weekday: 2, short: "Mar", long: "Martes" },
  { weekday: 3, short: "Mié", long: "Miércoles" },
  { weekday: 4, short: "Jue", long: "Jueves" },
  { weekday: 5, short: "Vie", long: "Viernes" },
  { weekday: 6, short: "Sáb", long: "Sábado" },
  { weekday: 0, short: "Dom", long: "Domingo" },
] as const;

const CLOCK = /^([01]\d|2[0-3]):[0-5]\d$/;

/** Una franja es válida si las dos horas son "HH:MM" y la segunda es posterior. */
export function isValidRange({ start, end }: Range): boolean {
  return CLOCK.test(start) && CLOCK.test(end) && start < end;
}

/** Repite cada franja en cada día elegido. Los días repetidos cuentan una vez. */
export function buildSlots(days: number[], ranges: Range[]): Slot[] {
  return [...new Set(days)]
    .sort(byWeekStart)
    .flatMap((weekday) => ranges.map(({ start, end }) => ({ weekday, start, end })));
}

/** Los índices de las dos primeras franjas que se pisan; las que solo se tocan no cuentan. */
export function findOverlap(ranges: Range[]): [number, number] | null {
  for (let i = 0; i < ranges.length; i++) {
    for (let j = i + 1; j < ranges.length; j++) {
      if (ranges[i].start < ranges[j].end && ranges[j].start < ranges[i].end) return [i, j];
    }
  }
  return null;
}

/**
 * Lo inverso de buildSlots, para editar un servicio guardado. Devuelve `null` cuando
 * los días no comparten las mismas franjas: ese horario no cabe en el formulario.
 */
export function toRanges(slots: Slot[]): Weekly | null {
  const byDay = new Map<number, string[]>();
  for (const { weekday, start, end } of slots) {
    byDay.set(weekday, [...(byDay.get(weekday) ?? []), `${start}-${end}`]);
  }

  const signatures = new Set([...byDay.values()].map((ranges) => ranges.sort().join("|")));
  if (signatures.size > 1) return null;

  const first = [...byDay.values()][0] ?? [];
  return {
    days: [...byDay.keys()].sort(byWeekStart),
    ranges: first.sort().map((range) => {
      const [start, end] = range.split("-");
      return { start, end };
    }),
  };
}

/** "Lun a Sáb", "Mar y Jue", "Lun, Mié y Vie". */
export function describeDays(days: number[]): string {
  const ordered = [...new Set(days)].sort(byWeekStart);
  const names = ordered.map((weekday) => WEEK.find((d) => d.weekday === weekday)?.short ?? "");
  if (names.length === 0) return "";
  if (names.length === 1) return names[0];

  const isRun = ordered.every(
    (weekday, i) => i === 0 || position(weekday) === position(ordered[i - 1]) + 1,
  );
  if (isRun && names.length > 2) return `${names[0]} a ${names[names.length - 1]}`;
  return `${names.slice(0, -1).join(", ")} y ${names[names.length - 1]}`;
}

function position(weekday: number): number {
  return WEEK.findIndex((d) => d.weekday === weekday);
}

function byWeekStart(a: number, b: number): number {
  return position(a) - position(b);
}

/** "Mar y Jue · 18:30 a 20:30", o "Horarios variables" si cambian según el día. Vacío sin tramos. */
export function describeSchedule(slots: Slot[]): string {
  if (slots.length === 0) return "";

  const weekly = toRanges(slots);
  if (weekly === null) return "Horarios variables";

  const ranges = weekly.ranges.map(({ start, end }) => `${start} a ${end}`).join(" y ");
  return `${describeDays(weekly.days)} · ${ranges}`;
}
