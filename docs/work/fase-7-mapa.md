# Fase 7 — Mapa propio con PMTiles (sin depender de OpenFreeMap)

Estado: **implementado y verificado en local** (Córdoba). Falta generar el archivo de producción y subirlo al bucket.

## Por qué

El fondo del mapa salía de OpenFreeMap, un servicio público y gratuito sin garantía de disponibilidad: si se cae o limita el tráfico, el mapa queda en blanco aunque la API ande perfecta. Con PMTiles el fondo es un solo archivo estático que servimos nosotros desde el bucket detrás de la CDN de la 7.4: sin tercero, sin cuota y sin servidor de tiles.

## Cómo funciona

- `frontend/lib/map.ts` arma el estilo con `@protomaps/basemaps` (tema `light`, etiquetas en español) sobre una fuente `pmtiles://<URL>`.
- `frontend/components/map/VectorTileLayer.tsx` registra el protocolo `pmtiles://` una vez y pasa el estilo a MapLibre. Lo usan los dos mapas (búsqueda y pin de la sede).
- **Sin `NEXT_PUBLIC_MAP_PMTILES_URL`** el mapa sigue usando OpenFreeMap (o `NEXT_PUBLIC_MAP_STYLE_URL`), como antes. El cambio es opt-in.
- La atribución cambia sola: con tiles propios dice "Protomaps / Datos de OpenStreetMap" (ODbL, obligatoria).

## Variables (se incrustan en el build: hay que volver a compilar y desplegar)

| Variable | Qué es | Ejemplo |
|---|---|---|
| `NEXT_PUBLIC_MAP_PMTILES_URL` | URL del archivo PMTiles; si falta, se usa OpenFreeMap | `https://cdn.ejemplo.com/map/argentina.pmtiles` |
| `NEXT_PUBLIC_MAP_ASSETS_URL` | Base de las tipografías y los íconos (por defecto `/map`) | `https://cdn.ejemplo.com/map` |

> **Windows con Git Bash:** al compilar, una variable que empieza con `/` (por ejemplo `/map/cordoba.pmtiles`) la convierte en una ruta de disco (`C:/Program Files/Git/map/...`) y el navegador pide `file:///...`. Compilar con `MSYS_NO_PATHCONV=1` o usar una URL completa `https://...`.

## Desarrollo local

```sh
scripts/map-assets.sh                      # tipografías e íconos -> frontend/public/map (git lo ignora)
# El archivo de la provincia (ver "Generar el archivo") a frontend/public/map/cordoba.pmtiles
echo 'NEXT_PUBLIC_MAP_PMTILES_URL=/map/cordoba.pmtiles' >> frontend/.env.local
cd frontend && npm run dev
```

`frontend/public/map/` está en el `.gitignore`: son ~75 MB (12 MB de tipografías, 208 KB de íconos y 66 MB del archivo de Córdoba) y no van al repositorio.

## Generar el archivo

Se usa el CLI oficial `pmtiles` (go-pmtiles, <https://github.com/protomaps/go-pmtiles/releases>) sobre una build diaria de Protomaps (`https://build.protomaps.com/AAAAMMDD.pmtiles`). `--dry-run` calcula el tamaño sin descargar nada.

Medido el 2026-11-03, hasta zoom 14 (más cerca el mapa se agranda sin pedir tiles nuevos):

| Región | `--bbox` (oeste,sur,este,norte) | Tamaño |
|---|---|---|
| Provincia de Córdoba (piloto) | `-65.9,-35.1,-61.6,-29.3` | 66 MB |
| Argentina | `-73.6,-55.2,-53.5,-21.7` | ~1,1 GB |

```sh
pmtiles extract https://build.protomaps.com/20261102.pmtiles cordoba.pmtiles \
  --bbox=-65.9,-35.1,-61.6,-29.3 --maxzoom=14
```

Para el piloto alcanza Córdoba; el país entero se genera igual cuando haya negocios fuera de la provincia. El archivo se regenera para actualizar el mapa (los datos de OSM cambian de a poco: con hacerlo una o dos veces al año sobra).

## Producción (pendiente, va con la 8.4)

1. Subir al bucket público, junto con las imágenes: `map/<region>.pmtiles` y la carpeta que deja `scripts/map-assets.sh <carpeta>` (`fonts/`, `sprites/`).
2. El bucket o la CDN **tienen que aceptar pedidos por rango de bytes** (HTTP Range, respuesta `206`): sin eso PMTiles no funciona. S3, R2 y B2 lo soportan. Probar con `curl -s -D - -o /dev/null -H 'Range: bytes=0-16383' <URL>` y buscar `206`.
3. CORS: permitir el origen del sitio para `GET` y `HEAD` y el header `Range`, y exponer `Content-Range`, `Content-Length`, `ETag`.
4. Compilar el frontend con las dos variables apuntando a la CDN.

## Verificación hecha (2026-11-03)

- `npm run lint`, `typecheck` y `format:check` en verde.
- El servidor de producción de Next responde `206 Partial Content` con `Content-Range` correcto para el archivo y sirve tipografías e íconos.
- Chromium sin interfaz abrió `/map` con el build de tiles propios: **todos los pedidos fueron a `localhost`** (ninguno a un tercero), el archivo se leyó con 10 pedidos parciales (`206`) y el mapa se dibujó completo (calles, parques, etiquetas en español y los pines de las sedes de Río Tercero).
- Hallazgo ajeno a este cambio: `GET /owners` responde 404 (un enlace del menú apunta a una ruta que no tiene página).

## No verificado

- Un bucket o CDN reales con Range y CORS (en local lo sirve Next).
- El mapa de ubicación de la sede (`BranchMap`): usa el mismo componente y el mismo estilo, pero no lo abrí.
- Visual en móvil.
