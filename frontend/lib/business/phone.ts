// WhatsApp de Argentina. El dueño escribe su número como lo marca en el teléfono,
// sin 0 ni 15 (10 dígitos: característica + abonado); la API guarda el formato de
// wa.me: código de país 54 + 9 (móvil) + los 10 dígitos.

const MOBILE_PREFIX = "549";
const LOCAL_DIGITS = 10;

/** Los dígitos del número, sin espacios, guiones ni el 0 inicial de la característica. */
export function localDigits(input: string): string {
  return input.replace(/\D/g, "").replace(/^0/, "");
}

/** El número en formato wa.me, o `null` si no tiene los 10 dígitos esperados. */
export function toWhatsApp(input: string): string | null {
  const digits = localDigits(input);
  return digits.length === LOCAL_DIGITS ? MOBILE_PREFIX + digits : null;
}

/** El número guardado, como se muestra y edita: "11 5824 9912". Vacío si no es un móvil argentino. */
export function fromWhatsApp(stored: string): string {
  if (!stored.startsWith(MOBILE_PREFIX)) return "";
  const digits = stored.slice(MOBILE_PREFIX.length);
  if (digits.length !== LOCAL_DIGITS) return digits;
  return `${digits.slice(0, 2)} ${digits.slice(2, 6)} ${digits.slice(6)}`;
}
