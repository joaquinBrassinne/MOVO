// El código del cupón como lo escribe una persona. Mismas reglas que `NormalizeCode` del
// backend (SDD §5.2): 8 símbolos de un alfabeto sin 0, O, 1 ni I, que la gente confunde.

export const CODE_LENGTH = 8;
const ALPHABET = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789";

/**
 * El código guardado ("ABCDEFGH") de lo que se escribió, o `null` si no puede ser un
 * código: así no se hace una consulta con algo que nunca va a existir.
 */
export function normalizeCode(raw: string): string | null {
  let code = "";
  for (const char of raw.toUpperCase()) {
    if (char === " " || char === "-") continue;
    if (!ALPHABET.includes(char)) return null;
    code += char;
  }
  return code.length === CODE_LENGTH ? code : null;
}

/** Lo que muestra el campo mientras se escribe: mayúsculas, sin espacios y "ABCD-EFGH". */
export function displayCode(raw: string): string {
  const symbols = raw
    .toUpperCase()
    .replace(/[^A-Z0-9]/g, "")
    .slice(0, CODE_LENGTH);
  return symbols.length > 4 ? `${symbols.slice(0, 4)}-${symbols.slice(4)}` : symbols;
}
