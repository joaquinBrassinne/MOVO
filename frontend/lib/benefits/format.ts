// Cómo se escribe un beneficio, un nombre y un momento en el panel. Funciones puras.

import { BENEFIT_TYPE, type ClaimedBenefit } from "./types.ts";

/** "2000" se escribe "2.000": los pesos con punto de miles, como en Argentina. */
export function formatPesos(amount: number): string {
  return String(amount).replace(/\B(?=(\d{3})+(?!\d))/g, ".");
}

/** El título de una tarjeta: "15% de descuento", "$2.000 de descuento" o el título propio. */
export function benefitHeadline(benefit: ClaimedBenefit): string {
  switch (benefit.type) {
    case BENEFIT_TYPE.PERCENT:
      return `${benefit.percent}% de descuento`;
    case BENEFIT_TYPE.AMOUNT:
      return `$${formatPesos(benefit.amount_ars ?? 0)} de descuento`;
    default:
      return benefit.title ?? "Beneficio";
  }
}

/** Lo que vale el beneficio, corto, para una frase: "15 %", "$2.000" o el título. */
export function benefitWorth(benefit: ClaimedBenefit): string {
  switch (benefit.type) {
    case BENEFIT_TYPE.PERCENT:
      return `${benefit.percent} %`;
    case BENEFIT_TYPE.AMOUNT:
      return `$${formatPesos(benefit.amount_ars ?? 0)}`;
    default:
      return benefit.title ?? "Beneficio";
  }
}

/** La inicial en mayúscula del nombre; "?" si no hay ninguna letra. */
export function initialOf(name: string): string {
  const letter = [...name.trim()].find((char) => /\p{L}/u.test(char));
  return letter ? letter.toLocaleUpperCase("es-AR") : "?";
}

/** "Lucía Martínez" se muestra "Lucía M.": el nombre y la inicial del apellido. */
export function shortName(name: string): string {
  const [first, ...rest] = name.trim().split(/\s+/);
  if (!first) return "";
  const last = rest.at(-1);
  return last ? `${first} ${initialOf(last)}.` : first;
}

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

/** "hace 2 min", "hace 3 h", "ayer", "hace 4 días"; más de una semana, la fecha corta. */
export function relativeTime(iso: string, now: Date): string {
  const then = new Date(iso);
  const elapsed = now.getTime() - then.getTime();
  if (Number.isNaN(elapsed)) return "";

  if (elapsed < MINUTE) return "recién";
  if (elapsed < HOUR) return `hace ${Math.floor(elapsed / MINUTE)} min`;
  if (elapsed < DAY) return `hace ${Math.floor(elapsed / HOUR)} h`;
  if (elapsed < 2 * DAY) return "ayer";
  if (elapsed < 7 * DAY) return `hace ${Math.floor(elapsed / DAY)} días`;
  return then.toLocaleDateString("es-AR", { day: "numeric", month: "short" });
}

/** "vence en 31 h" / "vence en 25 min"; ya vencido, "venció". */
export function timeLeft(iso: string, now: Date): string {
  const left = new Date(iso).getTime() - now.getTime();
  if (Number.isNaN(left)) return "";
  if (left <= 0) return "venció";
  if (left < HOUR) return `vence en ${Math.max(1, Math.floor(left / MINUTE))} min`;
  return `vence en ${Math.floor(left / HOUR)} h`;
}
