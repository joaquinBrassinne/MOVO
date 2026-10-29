const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

function plural(value: number, one: string, many: string): string {
  return `hace ${value} ${value === 1 ? one : many}`;
}

/** "hace 3 días", "hace 1 semana": the age of a review for the public list. */
export function relativeTime(iso: string, now: Date = new Date()): string {
  const elapsed = now.getTime() - new Date(iso).getTime();
  if (!Number.isFinite(elapsed) || elapsed < MINUTE) return "hace un momento";
  if (elapsed < HOUR) return plural(Math.floor(elapsed / MINUTE), "minuto", "minutos");
  if (elapsed < DAY) return plural(Math.floor(elapsed / HOUR), "hora", "horas");
  const days = Math.floor(elapsed / DAY);
  if (days < 7) return plural(days, "día", "días");
  if (days < 30) return plural(Math.floor(days / 7), "semana", "semanas");
  if (days < 365) return plural(Math.floor(days / 30), "mes", "meses");
  return plural(Math.floor(days / 365), "año", "años");
}
