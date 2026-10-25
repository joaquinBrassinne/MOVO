# Fase 3 — Búsqueda, mapa y contacto por WhatsApp

Rama: `phase-3/search-map-contact`. El estado general sigue siendo `docs/sdd.md`.

## Decisiones
- `visible_branches` nace mínima en 3.1 (sede activa de negocio `active`); la 5.4 la extiende con `CREATE OR REPLACE VIEW` sin cambiar columnas.
- `featured` es `false` en la vista hasta la fase 5, así que `/home/featured` responde vacío. Decidido para no reescribir nada después.
- La rotación de destacados usa una semilla que cambia cada 15 minutos (`RotationWindow`) y un hash del id del local.
- Sede principal: anotada como pendiente post-MVP (SDD §13, punto 9).

## 3.4 — Medición con 10k locales
`make perf-search` siembra 10 000 negocios (60 % en un área metropolitana, el resto repartido por el país; 10 % pausados, 5 % de sedes inactivas, 5 % piloto como sustituto de Pro) y mide 400 consultas por escenario, una tras otra. Presupuesto: p95 < 300 ms.

| Escenario | p50 | p95 | máx |
|---|---|---|---|
| `/search` 5 km, zona densa | 1,6 ms | 2,7 ms | 6,6 ms |
| `/search` 50 km, zona densa | 32,7 ms | 39,6 ms | 76 ms |
| `/search` 50 km, centro de la zona densa | 32,9 ms | 35,6 ms | 81 ms |
| `/search` 5 km + categoría | 1,2 ms | 2,1 ms | 3,7 ms |
| `/search` 5 km, zona rala | 0,5 ms | 1,0 ms | 1,6 ms |
| `/home/featured` 5 km | 1,1 ms | 1,7 ms | 2,7 ms |

Medido en un Postgres 16 + PostGIS 3.6 en Docker Desktop sobre Windows, con la base ya caliente. Cuenta la latencia de la consulta, no la de red ni la de HTTP.

### Hallazgo: radios amplios con la consulta directa sobre la vista
Con `ST_DWithin` aplicado dentro de la vista, PostGIS estima 1 fila para radios de 30 km o más en la zona densa y el planner elige un join con un `Seq Scan` de `businesses` por cada sede: **5 s** con 50 km (`EXPLAIN ANALYZE`: 29 millones de filas descartadas por el filtro del join). Subir `STATISTICS` de la columna no lo arregló, y forzar el plan genérico tampoco.

Qué lo ocultaba: la primera medición con orígenes aleatorios daba p95 de 40 ms, porque en esos puntos la estimación salía distinta. Solo `EXPLAIN` en el centro exacto de la zona densa mostró el problema. Por eso el escenario "centro de la zona densa" quedó fijo en la medición.

Solución: el filtro espacial corre primero sobre `branches` en un CTE `MATERIALIZED` y el resultado se une a `visible_branches` por id. La visibilidad sigue saliendo solo de la vista. Con eso el peor caso (50 km) baja a ~34 ms y el plan queda en búsquedas por clave primaria. Aplica a `/search` y a `/home/featured`.

### Pendiente de vigilar
- Con 10k locales alcanza. Si el volumen crece mucho (órdenes de magnitud) hay que repetir la medición; el `LIMIT` se aplica después de ordenar todo lo que está dentro del radio.
- La 5.4 reemplaza la vista: repetir `make perf-search` después, porque agrega un join con `subscriptions`.
