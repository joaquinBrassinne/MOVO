// Los números de página que muestra el listado.

/**
 * Hasta `size` páginas alrededor de la actual, sin salirse de 1..total. Con 10 páginas y la
 * 5 actual y tamaño 5 da 3, 4, 5, 6, 7.
 */
export function pageWindow(current: number, total: number, size = 5): number[] {
  if (total <= 0) return [];

  const length = Math.min(size, total);
  const first = Math.min(Math.max(1, current - Math.floor(length / 2)), total - length + 1);
  return Array.from({ length }, (_, i) => first + i);
}

/** Cuántas páginas hacen falta para `total` elementos. */
export function pageCount(total: number, pageSize: number): number {
  return Math.ceil(total / pageSize);
}
