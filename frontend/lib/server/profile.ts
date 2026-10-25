// Lectura de la ficha pública desde el servidor de Next (render SSR y metadatos).
// Va con caché corta: la ficha es la misma para todos y no vale pedirla dos veces (metadatos
// y página) ni por visita. El vencimiento queda bien por debajo de la vida de las URLs
// prefirmadas de las fotos, y el "abierto ahora" se refresca al minuto.

import { cacheLife } from "next/cache";

import { weekdayInCordoba, type Profile } from "@/lib/search/profile";

const API_URL = process.env.API_URL ?? "http://localhost:8080";
const TIMEOUT_MS = 10_000;

/** Un slug válido es minúsculas, números y guiones: cualquier otra cosa no llega a la API. */
const SLUG = /^[a-z0-9]+(-[a-z0-9]+)*$/;

export interface CachedProfile {
  profile: Profile;
  /** Día de la semana (0 = domingo, en Córdoba) en que se leyó: la hora de "ahora" solo se puede tomar dentro de lo cacheado. */
  today: number;
}

/** La ficha de un local público, o `null` si no existe o no es público. Falla si la API cae. */
export async function getProfile(slug: string): Promise<CachedProfile | null> {
  "use cache";
  cacheLife({ stale: 30, revalidate: 60, expire: 300 });

  if (!SLUG.test(slug)) return null;

  const response = await fetch(`${API_URL}/api/v1/businesses/${slug}`, {
    signal: AbortSignal.timeout(TIMEOUT_MS),
  });
  if (response.status === 404) return null;
  if (!response.ok) throw new Error(`profile of ${slug}: the API answered ${response.status}`);
  return { profile: (await response.json()) as Profile, today: weekdayInCordoba(new Date()) };
}
