// Precios en pesos enteros, como los escribe el dueño ("$ 5.500", "12000").

export const MAX_PRICE = 100_000_000;

/**
 * Los pesos enteros de lo que se escribió: `null` si quedó vacío (el precio es
 * opcional), `undefined` si no es un precio válido.
 */
export function parsePrice(text: string): number | null | undefined {
  const digits = text.replace(/[\s.$]/g, "");
  if (digits === "") return null;
  if (!/^\d+$/.test(digits)) return undefined;

  const amount = Number(digits);
  return amount > MAX_PRICE ? undefined : amount;
}

/** "5500" se muestra como "$ 5.500". */
export function formatPrice(amount: number): string {
  return `$ ${amount.toLocaleString("es-AR")}`;
}
