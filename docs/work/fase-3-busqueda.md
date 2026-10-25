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

## Cierre de la Fase 3 (DoD)

Verificado el 25/10/2026 sobre la rama `phase-2/photo-gallery`, que contiene todo lo de la fase (PRs #13, #14 y #15 más la mitad backend de 2.12).

| Criterio del SDD | Resultado | Cómo se comprobó |
|---|---|---|
| Búsqueda p95 < 300 ms | **Cumple.** Peor caso: 47 ms (50 km, centro denso). A 5 km: 2,6 ms | `make perf-search`: 10 000 negocios, 400 consultas por escenario |
| La ficha se renderiza del lado del servidor | **Cumple** | `curl /locales/<slug>` devuelve en el HTML el título, los horarios y el botón; no depende de JavaScript |
| El botón de WhatsApp abre la conversación y suma el contador | **Cumple** | `GET /api/v1/r/wa/{id}` por el proxy responde `302` a `wa.me` con mensaje prearmado, `no-store` y `no-referrer`; `whatsapp_click` pasó de 1 a 2 |
| Vistas de perfil (agregado de la fase) | **Cumple** | Una carga con user-agent de navegador suma 1 `profile_view` (un solo `POST`); el navegador sin interfaz (`HeadlessChrome`) no suma, por el filtro de bots |

Equivalente de `make ci` (el target no existe): `make fmt-check lint build`, `go test -count=1 ./...` (31 paquetes, todos ok, con Docker), ida y vuelta de las 13 migraciones (arriba, 13 abajo, arriba), y en `frontend` `format:check`, `eslint`, `typecheck`, `npm test` (65) y `build`.

### Tareas de la fase
3.1 a 3.9 y 3.8b hechas. Se sumaron en el camino: baja lógica de sedes (§4), categoría `trekking-outdoor`, 2.11 (horarios cortados), `POST /businesses/{slug}/view` y `make seed-dev`.

### Sin verificar o con límites conocidos
- Los e2e del panel (`e2e/04-panel`) no corren en esta máquina: el login agota el tiempo con y sin los cambios de la fase. El spec nuevo `e2e/06-search` sí pasa.
- La ficha de un local inexistente responde `200` con `noindex` y no `404` (streaming con Cache Components).
- La ficha cachea la lectura un minuto, así que "abierto ahora" puede ir hasta un minuto atrasado.
- Falta un spec e2e de la ficha (el render es del servidor y los mocks del navegador no lo cubren).

### Para planear la Fase 4
- **Vistas** (pedir la imagen con su nombre exacto al llegar a cada tarea): Beneficios y cupo mensual, Panel de inicio, Canjear cupón (4.10); Obtener cupón, Cupón generado, Mis cupones (4.11). La captura "cupón" de la ficha ya existe y corresponde a "Obtener cupón" (banner con el código y botón "Obtener cupón"); la ficha de hoy es su variante sin cupones.
- **Dependencias nuevas** (QR): generador y lector, con versión exacta y aprobación antes de instalar (nota en la Fase 4 del SDD).
- **Worker:** 4.8 (marcado diario de vencidos) es la primera tarea que necesita `cmd/worker`. Ahí también va a correr `PurgeObjects` (reintento de borrado de fotos, 2.12) y, más adelante, la 6.7.
- **Proxy del frontend:** hoy deja pasar `me/`, `admin/`, `plans`, `categories`, `search`, `home/featured`, el aviso de vista y `r/wa`. Fase 4 suma `access/{token}` y la generación de cupones (públicos, sin token de dueño).
- **Anti-abuso de cupones (4.5)** y **anti-inflado de métricas (6.3b)** comparten la necesidad de un limitador en memoria por IP: conviene una sola interfaz (lista para Redis).
- **Pendientes que tocan la fase:** sede principal (§13, punto 9) para la ficha y la tarjeta; repetir `make perf-search` cuando la 5.4 reemplace la vista; galería de fotos del panel (2.12, falta la pantalla "Fotos").
