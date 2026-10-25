// Configuración común de los mapas (SDD §12.3). Los tiles públicos de OSM son para el
// piloto; en producción la URL cambia por NEXT_PUBLIC_MAP_TILES_URL sin tocar código.
// Cada mapa (pin de la sede, Mapa de búsqueda) lee estas constantes, no su propia copia.

export const TILES_URL =
  process.env.NEXT_PUBLIC_MAP_TILES_URL ?? "https://tile.openstreetmap.org/{z}/{x}/{y}.png";

export const TILES_ATTRIBUTION =
  '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>';
