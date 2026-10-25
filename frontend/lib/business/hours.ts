// Horarios de atención de una sede: una franja por día, con un interruptor de abierto.
// La API guarda tramos sueltos; acá se pasa de un lado al otro.

import { isValidRange, WEEK } from "./schedule.ts";
import type { Slot } from "./types.ts";

export interface DayHours {
  weekday: number;
  open: boolean;
  start: string;
  end: string;
}

/** La franja que se propone al abrir un día que estaba cerrado. */
const DEFAULT_RANGE = { start: "09:00", end: "18:00" };

/**
 * Los siete días en el orden de la semana. `null` si algún día tiene más de un tramo
 * (turno cortado): ese horario no cabe en el formulario y se deja como está.
 */
export function toDayHours(slots: Slot[]): DayHours[] | null {
  const byDay = new Map<number, Slot[]>();
  for (const slot of slots) byDay.set(slot.weekday, [...(byDay.get(slot.weekday) ?? []), slot]);
  if ([...byDay.values()].some((day) => day.length > 1)) return null;

  return WEEK.map(({ weekday }) => {
    const slot = byDay.get(weekday)?.[0];
    return slot
      ? { weekday, open: true, start: slot.start, end: slot.end }
      : { weekday, open: false, ...DEFAULT_RANGE };
  });
}

/** Los tramos que se guardan: solo los días abiertos. */
export function toSlots(rows: DayHours[]): Slot[] {
  return rows.filter((row) => row.open).map(({ weekday, start, end }) => ({ weekday, start, end }));
}

/** "Copiar Lun a Vie": el horario del lunes pasa a martes, miércoles, jueves y viernes. */
export function copyMondayToWeekdays(rows: DayHours[]): DayHours[] {
  const monday = rows.find((row) => row.weekday === 1);
  if (!monday) return rows;

  return rows.map((row) =>
    row.weekday >= 2 && row.weekday <= 5
      ? { ...row, open: monday.open, start: monday.start, end: monday.end }
      : row,
  );
}

/** El día de la semana del primer día abierto con una franja mal armada, o `null` si todo está bien. */
export function firstInvalidDay(rows: DayHours[]): number | null {
  return rows.find((row) => row.open && !isValidRange(row))?.weekday ?? null;
}
