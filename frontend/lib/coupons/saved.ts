// Los cupones que el cliente guardó en este dispositivo. No hay cuenta: lo único que tiene es
// el enlace de acceso de cada cupón, y acá se guarda su token. Es una comodidad del navegador:
// si se borra, el cupón se recupera con su enlace.

const KEY = "movo.coupons";

/** El token que genera la API: 32 bytes en base64 de URL. Cualquier otra cosa no es un token. */
const TOKEN = /^[A-Za-z0-9_-]{43}$/;

export interface SavedCoupon {
  token: string;
  /** Cuándo se guardó, en RFC 3339. */
  savedAt: string;
}

/** Lo guardado, sin lo que no es un token ni los repetidos; lo más nuevo primero. */
export function parseSaved(raw: string | null): SavedCoupon[] {
  if (!raw) return [];

  let data: unknown;
  try {
    data = JSON.parse(raw);
  } catch {
    return [];
  }
  if (!Array.isArray(data)) return [];

  const seen = new Set<string>();
  const saved: SavedCoupon[] = [];
  for (const item of data) {
    const { token, savedAt } = (item ?? {}) as Partial<SavedCoupon>;
    if (typeof token !== "string" || !TOKEN.test(token) || seen.has(token)) continue;
    seen.add(token);
    saved.push({ token, savedAt: typeof savedAt === "string" ? savedAt : "" });
  }
  return saved.sort((a, b) => b.savedAt.localeCompare(a.savedAt));
}

export function isToken(value: string): boolean {
  return TOKEN.test(value);
}

export function readSaved(): SavedCoupon[] {
  try {
    return parseSaved(window.localStorage.getItem(KEY));
  } catch {
    return [];
  }
}

/** Guarda el token (si ya estaba, no lo repite) y devuelve la lista nueva. */
export function saveToken(token: string, now = new Date()): SavedCoupon[] {
  const current = readSaved();
  if (!isToken(token) || current.some((item) => item.token === token)) return current;

  const next = [{ token, savedAt: now.toISOString() }, ...current];
  try {
    window.localStorage.setItem(KEY, JSON.stringify(next));
  } catch {
    // Sin almacenamiento (modo privado, bloqueado): el cupón se ve por su enlace igual.
  }
  return next;
}

/** Lo guardado tal cual lo tiene el navegador (texto o `null`): una lectura estable para React. */
export function readRawSaved(): string | null {
  try {
    return window.localStorage.getItem(KEY);
  } catch {
    return null;
  }
}

/** Avisa cuando cambia lo guardado desde otra pestaña. En la misma pestaña, `saveToken` ya vuelve a leer. */
export function subscribeSaved(onChange: () => void): () => void {
  window.addEventListener("storage", onChange);
  return () => window.removeEventListener("storage", onChange);
}

export function forgetToken(token: string): SavedCoupon[] {
  const next = readSaved().filter((item) => item.token !== token);
  try {
    window.localStorage.setItem(KEY, JSON.stringify(next));
  } catch {
    // Ver arriba.
  }
  return next;
}
