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

- Fix al volver atrás: con `cacheComponents` la página queda oculta en `<Activity>` y Leaflet falla con "Map container is being reused". `MountWhileVisible` monta el mapa solo mientras se ve.
- Fix del mapa en blanco: MapLibre 6 deduce la URL del worker desde `import.meta.url` y Turbopack la rompe ("Worker failed to load"). `setWorkerUrl(new URL("maplibre-gl/dist/maplibre-gl-worker.mjs", import.meta.url))`.
- Probado a mano en el navegador (2026-10-31): el mapa de búsqueda funciona, incluido volver atrás.

## Pendiente
- Revisar a mano el pin de la sede (onboarding); atribución visible.
- Confirmar la política de uso de OpenFreeMap antes de lanzar.
- Correr e2e (Playwright) con la capa nueva.
- Migrar a Protomaps antes de producción.
