/** Junta las clases que valen (las condiciones falsas se descartan). Sin dependencias: no resuelve conflictos. */
export function cn(...parts: (string | false | null | undefined)[]): string {
  return parts.filter(Boolean).join(" ");
}
