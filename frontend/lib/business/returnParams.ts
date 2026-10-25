// Lo que Mercado Pago agrega a la URL de retorno (por ejemplo `plan`) se guarda solo como
// pista para el super admin: nunca es un dato confiable ni activa nada.

const STORAGE_KEY = "movo:mp-return-params";
const MAX_PARAMS = 10;
const MAX_KEY_LENGTH = 40;
const MAX_VALUE_LENGTH = 200;

export type ReturnParams = Record<string, string>;

/** Los parámetros de una query string, sin vacíos ni valores fuera de tamaño. */
export function parseReturnParams(search: string): ReturnParams {
  const params: ReturnParams = {};
  for (const [rawKey, rawValue] of new URLSearchParams(search)) {
    const key = rawKey.trim();
    const value = rawValue.trim();
    if (!key || !value || key.length > MAX_KEY_LENGTH || value.length > MAX_VALUE_LENGTH) continue;
    if (Object.keys(params).length >= MAX_PARAMS) break;
    params[key] = value;
  }
  return params;
}

/** Guarda los parámetros de la URL actual para el envío, que ocurre varias pantallas después. */
export function rememberReturnParams(search: string): void {
  const params = parseReturnParams(search);
  if (Object.keys(params).length === 0) return;
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(params));
  } catch {
    // Sin almacenamiento disponible solo se pierde la pista: el alta sigue igual.
  }
}

/** Los parámetros guardados al volver de Mercado Pago, o ninguno. */
export function recalledReturnParams(): ReturnParams {
  try {
    const stored: unknown = JSON.parse(sessionStorage.getItem(STORAGE_KEY) ?? "{}");
    if (typeof stored !== "object" || stored === null) return {};
    // El servidor vuelve a acotar el tamaño: acá solo se descarta lo que no es texto.
    return Object.fromEntries(Object.entries(stored).filter(([, v]) => typeof v === "string"));
  } catch {
    return {};
  }
}
