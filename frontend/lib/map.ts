// Configuración común de los mapas (SDD §12.3). Los tiles públicos de OSM son para el
// piloto; en producción la URL cambia por NEXT_PUBLIC_MAP_TILES_URL sin tocar código.
// Cada mapa (pin de la sede, Mapa de búsqueda) lee estas constantes, no su propia copia.
//
// Los tiles públicos de OSM no tienen disponibilidad garantizada y bloquean con un 403
// ("Access blocked") a las IP con mucho tráfico, sobre todo desde localhost. Si aparece:
//   - en desarrollo, apuntá NEXT_PUBLIC_MAP_TILES_URL en `frontend/.env.local` a otro servidor;
//   - en producción, cambiala por un proveedor o por Protomaps (SDD §12.3). La variable se
//     incrusta en el build: hay que volver a desplegar.

export const TILES_URL =
  process.env.NEXT_PUBLIC_MAP_TILES_URL ?? "https://tile.openstreetmap.org/{z}/{x}/{y}.png";

export const TILES_ATTRIBUTION =
  '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>';
