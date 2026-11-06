// Cómo se escribe el tiempo de un cupón. Funciones puras: se prueban sin React.

const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
/** Argentina no tiene horario de verano: es UTC-3 todo el año (SDD §5.2, el cupo se cuenta así). */
const ARGENTINA_OFFSET = -3 * HOUR;

/** Lo que falta para `iso` como "47:59" (horas y minutos); "00:00" si ya pasó o no es una fecha. */
export function formatCountdown(iso: string, now: Date): string {
  const left = new Date(iso).getTime() - now.getTime();
  if (Number.isNaN(left) || left <= 0) return "00:00";

  const hours = Math.floor(left / HOUR);
  const minutes = Math.floor((left % HOUR) / MINUTE);
  return `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}`;
}

/** "Vence en 31 hs" / "Vence en 25 min": el resumen corto de las tarjetas. */
export function formatExpiresIn(iso: string, now: Date): string {
  const left = new Date(iso).getTime() - now.getTime();
  if (Number.isNaN(left) || left <= 0) return "Vencido";
  if (left < HOUR) return `Vence en ${Math.max(1, Math.floor(left / MINUTE))} min`;
  return `Vence en ${Math.floor(left / HOUR)} hs`;
}

/** "24 de oct": el día de un hecho, en hora de Argentina. */
export function formatShortDate(iso: string): string {
  const moment = new Date(new Date(iso).getTime() + ARGENTINA_OFFSET);
  if (Number.isNaN(moment.getTime())) return "";
  const months = [
    "ene",
    "feb",
    "mar",
    "abr",
    "may",
    "jun",
    "jul",
    "ago",
    "sep",
    "oct",
    "nov",
    "dic",
  ];
  return `${moment.getUTCDate()} de ${months[moment.getUTCMonth()]}`;
}

/** El día en que el cupo mensual de los locales vuelve a empezar ("AAAA-MM-01"), en hora de Argentina. */
export function nextQuotaReset(now: Date): string {
  const local = new Date(now.getTime() + ARGENTINA_OFFSET);
  const next = new Date(Date.UTC(local.getUTCFullYear(), local.getUTCMonth() + 1, 1));
  return `${next.getUTCFullYear()}-${String(next.getUTCMonth() + 1).padStart(2, "0")}-01`;
}
