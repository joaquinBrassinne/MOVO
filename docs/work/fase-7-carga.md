# Fase 7 — Prueba de carga y ajuste del pool (tarea 7.5)

Estado: **herramientas listas, mediciones pendientes**. Las celdas de las tablas están vacías a propósito: se completan con corridas reales, no con estimaciones. El DoD de la fase (mejora medible contra la línea base de la Fase 3) se cierra recién cuando estén completas.

Línea base de la Fase 3 (`docs/work/fase-3-busqueda.md`, solo la consulta, sin red ni HTTP, 10 000 negocios): `/search` 5 km zona densa p95 **2,7 ms**; 50 km centro denso p95 **35,6 ms** (máximo 81 ms). El presupuesto es p95 < 300 ms.

## Qué mide

`backend/loadtest/search.js` (k6) pega a `GET /api/v1/search` con `lat`, `lng`, `radius_m` (5 km en 4 de cada 5 consultas, 50 km en la quinta) y `limit=21`, desde ~100 orígenes repartidos en una grilla de ±0,09° alrededor del centro de la zona densa de `make seed-dev` (Río Tercero, `-32.1738, -64.1146`).

Umbrales (si fallan, k6 sale con código distinto de cero):

- `http_req_duration{name:search}` p(95) < 300 ms.
- `http_req_failed` < 1 %.

Las etapas son: 30 s de subida a un quinto del pico, 1 min hasta el pico, 2 min sostenidos y 30 s de bajada. El pico es `PEAK_VUS` (50 por defecto).

## Cómo correrlo

Requisitos: k6 instalado (`winget install k6 --accept-source-agreements` en Windows) y la base con datos de `make seed-dev`.

```sh
make up && make migrate-up && make seed-dev
make run-api            # en otra terminal, la API en :8080
make load-search        # BASE_URL=http://localhost:8080 por defecto
```

Cambiar el pico o el destino sin tocar el script:

```sh
PEAK_VUS=100 BASE_URL=http://localhost:8080 make load-search
```

`make load-search` sin k6 instalado imprime cómo instalarlo y sale con error; no falla en silencio.

### Comparar caché apagada contra encendida

La caché es de la API, no de k6: se controla con `REDIS_URL` en el entorno del proceso `api`. Mismo script, mismos datos, mismo pool; solo cambia esa variable.

1. **Caché apagada (línea base de la fase):** API sin `REDIS_URL`. Correr `make load-search` y anotar la fila "sin caché".
2. **Caché encendida:** reiniciar la API con `REDIS_URL=redis://localhost:6379` (Redis levantado, por ejemplo con `docker compose`). Dejar un minuto de calentamiento (una corrida corta, sin anotar) para llenar las celdas, y después correr `make load-search` y anotar la fila "con caché".
3. Repetir cada escenario al menos 2 veces y anotar la mediana; una sola corrida no alcanza para comparar.

Como el TTL es 1 minuto (`SEARCH_CACHE_TTL`), una corrida de 4 minutos con caché encendida mide hits y misses mezclados. Anotar en la tabla la tasa de aciertos si se puede leer de Redis (`INFO stats`, `keyspace_hits` y `keyspace_misses` antes y después de la corrida).

Mientras se mide, no escribir datos (altas, cupones, invalidaciones de la 7.2): cada escritura sube la generación y vacía la caché, y el resultado deja de ser comparable.

## Tabla de resultados

Completar con mediciones reales. Una fila por corrida (la mediana de las repeticiones).

| Fecha | Caché (`REDIS_URL`) | `DB_MAX_CONNS` | `DB_MIN_CONNS` | VUs pico | p50 (ms) | p95 (ms) | rps | `http_req_failed` | Aciertos de caché | Notas |
|---|---|---|---|---|---|---|---|---|---|---|
| | sin | | | | | | | | — | |
| | con | | | | | | | | | |
| | sin | | | | | | | | — | |
| | con | | | | | | | | | |

Los valores de `p50`, `p95` y `rps` salen del resumen de k6: `http_req_duration` (med y p(95)) y `http_reqs` (rate). Las filas de pool sirven para el ajuste de la sección siguiente; las de caché, para la mejora de la 7.1 y 7.2.

## Pool de conexiones

### Variables

Leídas por `config.LoadPool` (`backend/internal/platform/config/pool.go`). Sin variable, el pool queda como lo dejaba pgx antes de esta tarea:

| Variable | Por defecto | Qué controla |
|---|---|---|
| `DB_MAX_CONNS` | mayor entre 4 y la cantidad de CPUs | Tope de conexiones abiertas por el proceso |
| `DB_MIN_CONNS` | 0 | Conexiones que se mantienen abiertas en reposo; no puede superar `DB_MAX_CONNS` |
| `DB_MAX_CONN_LIFETIME` | 1h | Edad máxima de una conexión antes de reemplazarla |
| `DB_MAX_CONN_IDLE_TIME` | 30m | Tiempo máximo ocioso antes de cerrarla |

Un valor inválido (no numérico, cero o negativo, mínimo mayor que máximo) hace fallar el arranque con un mensaje que nombra la variable.

Las variables se aplican por proceso: cada réplica de `cmd/api` y `cmd/worker` abre su propio pool, así que el total de conexiones a Postgres es la suma de `DB_MAX_CONNS` de todos los procesos. Revisar `max_connections` de Postgres (100 por defecto) antes de subir el tope.

### Procedimiento de ajuste

1. Medir la línea base con los valores por defecto (sin variables de pool) y registrar la fila.
2. Subir o bajar **un** valor por corrida (por ejemplo `DB_MAX_CONNS` de 4 a 10). No cambiar pool y caché a la vez.
3. Mirar tres señales en cada corrida: `http_req_duration` p(95), la espera por conexión (si el p95 sube con más VUs pero el tiempo de la consulta no, el pool es el cuello) y `pg_stat_activity` durante la carga (conexiones `idle in transaction` o en espera revelan un pool sobredimensionado o transacciones largas).
4. Dejar el valor que minimiza p95 sin subir el número de conexiones activas más allá de lo que el servidor de Postgres tolera. Si ningún valor mejora el p95, el cuello está en la consulta (ver la Fase 3, `EXPLAIN ANALYZE`), no en el pool.
5. Anotar el valor elegido y la corrida que lo justifica en la tabla, y en `docs/sdd.md` solo si cambia un default.

Nota: `DB_MAX_CONNS` pisa `pool_max_conns` de la URL de `DATABASE_URL`, porque `ConnectWithPool` aplica el valor del entorno después de parsear la URL. Mientras no haya URL con esos parámetros, no cambia nada.

## Verificación de lo que se entregó

- `go test ./internal/platform/config/` cubre defaults, lectura de valores, rechazo de valores inválidos y `DB_MIN_CONNS` por encima del `DB_MAX_CONNS` por defecto.
- `go test ./internal/platform/db/` cubre que `ConnectWithPool` rechaza una configuración imposible antes de conectar y que una configuración válida llega al ping y falla ahí contra un servidor inalcanzable.
- Los tests no prueban la corrida de k6 ni la medición: esa parte es manual, según el procedimiento de arriba.

## Pendiente

- Correr la tabla completa (caché apagada y encendida, al menos dos pools) y completar las celdas vacías.
- Cerrar el DoD de la fase con esos números, contra la línea base de la Fase 3.
- Repetir la medición de la Fase 3 (`make perf-search`) si cambia la consulta o el volumen de datos.
