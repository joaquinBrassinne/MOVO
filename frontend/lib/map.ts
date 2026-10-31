// Configuración común de los mapas (SDD §12.3). Los tiles son vectoriales (MapLibre GL
// dentro de Leaflet). El piloto usa OpenFreeMap; en producción el estilo cambia por
// NEXT_PUBLIC_MAP_STYLE_URL (por ejemplo Protomaps) sin tocar código.
// Cada mapa (pin de la sede, Mapa de búsqueda) lee estas constantes, no su propia copia.

export const MAP_STYLE_URL =
  process.env.NEXT_PUBLIC_MAP_STYLE_URL ?? "https://tiles.openfreemap.org/styles/liberty";

// Los estilos de OpenFreeMap y Protomaps usan datos de OpenStreetMap: la atribución es obligatoria.
export const MAP_ATTRIBUTION =
  '<a href="https://openfreemap.org" target="_blank" rel="noreferrer">OpenFreeMap</a> ' +
  '&copy; <a href="https://www.openmaptiles.org/" target="_blank" rel="noreferrer">OpenMapTiles</a> ' +
  'Datos de <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noreferrer">OpenStreetMap</a>';
