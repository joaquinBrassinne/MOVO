// El QR de un cupón. Lleva únicamente el código del cupón (SDD 4.11): ni el teléfono, ni el
// nombre, ni el enlace de acceso. Quien lo escanee solo obtiene el código, que por sí solo
// no sirve de nada fuera del negocio que lo emitió.

import { encode } from "uqr";

export interface QrShape {
  /** Lado de la grilla, borde incluido. */
  size: number;
  /** Los módulos oscuros como un solo trazo SVG, una tira por cada corrida de la fila. */
  path: string;
}

/** La forma del QR de `text`, lista para dibujar en un `<svg viewBox="0 0 size size">`. */
export function qrShape(text: string): QrShape {
  // Corrección "M": aguanta un 15 % de daño (una pantalla rayada) sin agrandar el código.
  const { data, size } = encode(text, { ecc: "M", border: 2 });

  const strips: string[] = [];
  data.forEach((row, y) => {
    let x = 0;
    while (x < size) {
      if (!row[x]) {
        x += 1;
        continue;
      }
      const start = x;
      while (x < size && row[x]) x += 1;
      strips.push(`M${start} ${y}h${x - start}v1h-${x - start}z`);
    }
  });

  return { size, path: strips.join("") };
}
