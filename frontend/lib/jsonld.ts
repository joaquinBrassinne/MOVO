/**
 * Serializa datos estructurados para incrustarlos en un `<script type="application/ld+json">`.
 * Escapa `<`, `>` y `&` para que un texto libre (nombre,
 * descripción, dirección) no pueda cerrar el script ni inyectar HTML.
 */
export function jsonLd(data: unknown): string {
  return JSON.stringify(data)
    .replace(/</g, "\\u003c")
    .replace(/>/g, "\\u003e")
    .replace(/&/g, "\\u0026");
}
