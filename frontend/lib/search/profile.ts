// Contrato de GET /businesses/{slug} y lo que la ficha hace con esos datos (horarios
// agrupados, estado "abierto ahora"). Sin React: se prueba con node:test.

import type { ClaimedBenefit } from "../benefits/types.ts";
import type { Slot } from "../business/types.ts";

export interface ProfileCategory {
  slug: string;
  name: string;
}

export interface OpenNow {
  open: boolean;
  closes_at: string | null;
  opens_at: string | null;
  opens_weekday: number | null;
}

export interface ProfileBranch {
  branch_id: number;
  name: string;
  address: string;
  city: string;
  reference: string;
  external_booking_url: string | null;
  lat: number;
  lng: number;
  rating_avg: number;
  rating_count: number;
  hours: Slot[];
  open_now: OpenNow;
}

export interface ProfileService {
  id: number;
  branch_id: number | null;
  name: string;
  description: string;
  price: number | null;
  schedule: Slot[];
}

export interface Profile {
  name: string;
  slug: string;
  description: string;
  category: ProfileCategory;
  verified: boolean;
  branches: ProfileBranch[];
  services: ProfileService[];
  /** Los beneficios que el local ofrece hoy (activos y no pausados), el mejor primero. */
  benefits: ClaimedBenefit[];
  images: string[];
}

/** Los días como los lee la interfaz: la semana empieza el lunes. */
const WEEK_ORDER = [1, 2, 3, 4, 5, 6, 0];
const LONG = ["Domingo", "Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado"];

/** "Sábados" para un día suelto del fin de semana, "Lunes" tal cual: así lo muestra el diseño. */
function single(weekday: number): string {
  return weekday === 0 || weekday === 6 ? `${LONG[weekday]}s` : LONG[weekday];
}

export interface HoursGroup {
  label: string;
  /** "08:00 – 12:00", una por franja; vacío cuando el grupo está cerrado. */
  ranges: string[];
}

/**
 * Agrupa los días consecutivos con las mismas franjas ("Lunes a Viernes") y muestra el
 * resto por separado. Un día sin franjas es "Cerrado": los demás días cerrados se juntan
 * igual que los abiertos.
 */
export function groupHours(slots: Slot[]): HoursGroup[] {
  const byDay = WEEK_ORDER.map((weekday) => ({
    weekday,
    ranges: slots
      .filter((slot) => slot.weekday === weekday)
      .sort((a, b) => a.start.localeCompare(b.start))
      .map((slot) => `${slot.start} – ${slot.end}`),
  }));

  const groups: { days: number[]; ranges: string[] }[] = [];
  for (const day of byDay) {
    const last = groups.at(-1);
    if (last && last.ranges.join("|") === day.ranges.join("|")) last.days.push(day.weekday);
    else groups.push({ days: [day.weekday], ranges: day.ranges });
  }

  return groups.map(({ days, ranges }) => ({
    label:
      days.length === 1
        ? single(days[0])
        : days.length === 2
          ? `${LONG[days[0]]} y ${LONG[days[1]]}`
          : `${LONG[days[0]]} a ${LONG[days.at(-1)!]}`,
    ranges,
  }));
}

/** El texto del estado de la sede: "Abierto ahora · cierra a las 12:00", "Cerrado · abre a las 14:00"… */
export function describeOpenNow(status: OpenNow, today: number): string {
  if (status.open && status.closes_at) return `Abierto ahora · cierra a las ${status.closes_at}`;
  if (status.opens_at === null || status.opens_weekday === null) return "Cerrado";
  if (status.opens_weekday === today) return `Cerrado · abre a las ${status.opens_at}`;
  return `Cerrado · abre el ${LONG[status.opens_weekday].toLowerCase()} a las ${status.opens_at}`;
}

/** El día de la semana (0 = domingo) en Córdoba en el instante `now`, sea cual sea la zona del servidor. */
export function weekdayInCordoba(now: Date): number {
  const name = new Intl.DateTimeFormat("en-US", {
    weekday: "short",
    timeZone: "America/Argentina/Cordoba",
  }).format(now);
  return ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"].indexOf(name);
}
