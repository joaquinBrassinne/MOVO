# Mapa: de tiles de OSM a OpenFreeMap

## Objetivo
Dejar de depender de los tiles públicos de OSM (la política de uso los bloquea, 403 en localhost) sin reescribir los mapas.

## Decisiones
- Se mantiene Leaflet/react-leaflet (marcadores, clustering y e2e `.leaflet-marker-icon` no cambian).
- Capa vectorial con `@maplibre/maplibre-gl-leaflet` 0.1.4 + `maplibre-gl` 6.16.0 (versiones exactas).
- Estilo piloto: OpenFreeMap `liberty` (sin API key, sin SLA). Variable `NEXT_PUBLIC_MAP_STYLE_URL` (reemplaza a `NEXT_PUBLIC_MAP_TILES_URL`).
- Producción: Protomaps (PMTiles en el bucket, SDD §12.3). Solo cambia la URL del estilo.

## Hecho
- `components/map/VectorTileLayer.tsx`, usado por `SearchMap` y `BranchMap`.
- `lib/map.ts`, `.env.example` y `docs/sdd.md` (§ stack, riesgos, 12.3) actualizados.
- typecheck, lint, format:check y build OK.

## Pendiente
- Revisar a mano en el navegador (`make up` + `npm run dev`): Mapa de búsqueda y pin de la sede; atribución visible.
- Confirmar la política de uso de OpenFreeMap antes de lanzar.
- Correr e2e (Playwright) con la capa nueva.
- Migrar a Protomaps antes de producción.
