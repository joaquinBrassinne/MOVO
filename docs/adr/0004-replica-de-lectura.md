# 0004 — No se agrega una réplica de lectura de PostgreSQL por ahora

- **Estado:** aceptada
- **Fecha:** 2026-11-03
- **Decide sobre:** SDD Fase 7, tarea 7.6

## Contexto

El SDD pide evaluar una réplica de lectura "solo si las métricas lo exigen". Una réplica es una segunda copia de PostgreSQL que atiende solo consultas de lectura para aliviar a la principal. Cuesta un servidor más, hay que vigilar el retraso de replicación (una lectura puede ver datos viejos) y complica los respaldos y el despliegue.

Las mediciones de la Fase 7 (`docs/work/fase-7-carga.md`, 2026-11-03, base de desarrollo con 15 sedes visibles, todo en una misma máquina de 12 núcleos):

- La API Go sirve la búsqueda pública a **~4 500 pedidos por segundo con un p95 de ~15 ms** y 0 % de errores, contra un presupuesto de 300 ms.
- El tamaño del pool de conexiones casi no importa: con 12 y con 40 conexiones el resultado es el mismo; con 4 baja un 22 %.
- El caché Redis (7.1 y 7.2) acierta el 99,9 % de las búsquedas pero a este volumen no mejora la latencia: Postgres ya responde más rápido que las idas a Redis.
- El cuello de botella real es el proxy de Next, que se satura cerca de los 1 000 pedidos por segundo. Está delante de la base: una réplica no lo toca.
- Línea base de la Fase 3 con 10 000 negocios (solo la consulta): peor caso, p95 de 47 ms.

## Decisión

No se agrega réplica de lectura. Postgres no es el límite del sistema: los límites medidos están en el proxy de Next, que se resuelve sin tocar la base (ver 8.4: el proxy HTTPS manda la búsqueda directo a la API Go).

## Alternativas y por qué no ahora

| Alternativa | Por qué no ahora |
|---|---|
| Réplica de lectura | Resuelve un problema que no existe: la base usa una fracción de su capacidad. Suma costo, otro punto de falla y datos viejos en las lecturas. |
| Más conexiones al pool | Medido: de 12 a 40 no cambia nada. |
| Más caché Redis | Ya está hecho (7.1, 7.2) y apagado por defecto; se enciende cuando el volumen lo justifique. |

## Cuándo reabrir esta decisión

Cualquiera de estas señales, sostenida en producción y no en una prueba de laboratorio, abre una nueva evaluación:

1. El p95 de `/search` medido en el servidor supera los 300 ms de forma sostenida.
2. La CPU de Postgres pasa del 70 % de forma sostenida, o las conexiones se acercan a `max_connections`.
3. Las lecturas pesadas (métricas del panel, reportes del super admin) empiezan a frenar las escrituras de los dueños.

Antes de una réplica, en este orden: encender el caché Redis, revisar `EXPLAIN ANALYZE` de las consultas lentas y subir el tamaño del servidor. La réplica va al final.

## Consecuencias

- **Positivas:** sin costo ni complejidad operativa extra para el piloto; el esfuerzo va al cuello de botella real (el proxy).
- **Negativas:** las mediciones se hicieron con 15 sedes y no con volumen real de producción, así que la decisión se confirma con las señales de arriba una vez que haya tráfico.
