// Orden de las fotos del local. La primera es la portada: elegirla o moverla es cambiar el orden.

/** Los ids con `id` un lugar antes (`-1`) o después (`1`), o `null` si ya está en el borde o no existe. */
export function moveInOrder(ids: number[], id: number, step: -1 | 1): number[] | null {
  const from = ids.indexOf(id);
  const to = from + step;
  if (from === -1 || to < 0 || to >= ids.length) return null;

  const next = [...ids];
  [next[from], next[to]] = [next[to], next[from]];
  return next;
}

/** Los ids con `id` al frente: así se elige la portada. `null` si ya lo es o no existe. */
export function coverFirst(ids: number[], id: number): number[] | null {
  if (ids.indexOf(id) <= 0) return null;
  return [id, ...ids.filter((other) => other !== id)];
}
