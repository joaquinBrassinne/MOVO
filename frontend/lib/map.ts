// Configuración común de los mapas (SDD §12.3). Los tiles son vectoriales (MapLibre GL
// dentro de Leaflet). Cada mapa (pin de la sede, Mapa de búsqueda) lee estas constantes, no su
// propia copia.
//
// Hay dos fuentes de fondo, y la variable de entorno decide cuál (se incrusta en el build: hay
// que volver a desplegar para cambiarla):
//   - NEXT_PUBLIC_MAP_PMTILES_URL: archivo PMTiles propio (Protomaps), alojado en nuestro bucket
//     detrás de la CDN. Sin terceros en producción. Las tipografías y los íconos salen de
//     NEXT_PUBLIC_MAP_ASSETS_URL (por defecto /map; ver docs/work/fase-7-mapa.md).
//   - Sin esa variable: el estilo público de OpenFreeMap, o el que diga NEXT_PUBLIC_MAP_STYLE_URL.
//     Sirve para el piloto y el desarrollo, pero no tiene disponibilidad garantizada.

import { layers, namedFlavor } from "@protomaps/basemaps";
import type { StyleSpecification } from "maplibre-gl";

const PMTILES_URL = process.env.NEXT_PUBLIC_MAP_PMTILES_URL;
const ASSETS_URL = process.env.NEXT_PUBLIC_MAP_ASSETS_URL ?? "/map";

export const MAP_STYLE_URL =
  process.env.NEXT_PUBLIC_MAP_STYLE_URL ?? "https://tiles.openfreemap.org/styles/liberty";

/** Si el fondo sale de nuestro propio archivo PMTiles. */
export const USES_OWN_TILES = Boolean(PMTILES_URL);

const SOURCE_ID = "protomaps";

/**
 * El estilo del mapa: el objeto de Protomaps (con el archivo PMTiles propio) o la URL del estilo
 * público. `origin` completa las rutas relativas: MapLibre las resuelve dentro de un worker, donde
 * una ruta sin dominio no es confiable.
 */
export function mapStyle(origin: string): string | StyleSpecification {
  if (!PMTILES_URL) {
    return MAP_STYLE_URL;
  }

  const assets = new URL(ASSETS_URL, origin).toString().replace(/\/$/, "");
  return {
    version: 8,
    glyphs: `${assets}/fonts/{fontstack}/{range}.pbf`,
    sprite: `${assets}/sprites/v4/light`,
    sources: {
      [SOURCE_ID]: {
        type: "vector",
        url: `pmtiles://${new URL(PMTILES_URL, origin).toString()}`,
      },
    },
    layers: layers(SOURCE_ID, namedFlavor("light"), { lang: "es" }),
  };
}

// Los datos son de OpenStreetMap (ODbL): la atribución es obligatoria con cualquier fuente.
const OSM =
  'Datos de <a href="https://www.openstreetmap.org/copyright" target="_blank" rel="noreferrer">OpenStreetMap</a>';

export const MAP_ATTRIBUTION = USES_OWN_TILES
  ? `<a href="https://protomaps.com" target="_blank" rel="noreferrer">Protomaps</a> ${OSM}`
  : '<a href="https://openfreemap.org" target="_blank" rel="noreferrer">OpenFreeMap</a> ' +
    '&copy; <a href="https://www.openmaptiles.org/" target="_blank" rel="noreferrer">OpenMapTiles</a> ' +
    OSM;
