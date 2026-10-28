import { randomBytes } from "node:crypto";

/**
 * Cabecera de un cliente nuevo para los límites por origen de la API. Todos los tests salen
 * de la misma máquina; sin esto compartirían el cupo de cupones de una sola dirección. La API
 * del E2E confía en el servidor de Next (TRUSTED_PROXIES), que reenvía esta cadena. Las
 * direcciones son del rango de documentación 2001:db8::/32 y no se repiten entre llamadas.
 */
export function freshClient(): Record<string, string> {
  const bytes = randomBytes(8);
  const words = [0, 2, 4, 6].map((offset) => bytes.readUInt16BE(offset).toString(16));
  return { "x-forwarded-for": `2001:db8:${words.join(":")}:0:1` };
}
