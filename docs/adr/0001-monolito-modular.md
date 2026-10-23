# 0001 — Monolito modular con dos binarios de servicio

- **Estado:** aceptada
- **Fecha:** 2026-10-23
- **Decide sobre:** SDD §3

## Contexto

MOVO arranca con locales de Almafuerte y Río Tercero: poco tráfico, un equipo chico y un dominio que todavía se está descubriendo (locales, beneficios, canjes, suscripciones). Las decisiones de arquitectura tienen que abaratar el cambio; la escala no es el problema.

El riesgo real no es de rendimiento, es de corrección: cobrar suscripciones, validar webhooks de Mercado Pago, canjear un código una sola vez y no filtrar datos personales de los dueños.

## Decisión

1. **Un repositorio, un despliegue, módulos con fronteras estrictas.** Un monolito modular, no microservicios: un módulo se puede extraer más adelante si aparece una razón concreta (carga, equipos, despliegue independiente). Diseñar preparado para separar alcanza; separar antes es costo sin retorno.
2. **Dos binarios de servicio y uno de migración, sobre la misma base de código:**
   - `cmd/api`: HTTP stateless con JWT, escalable detrás de un balanceador.
   - `cmd/worker`: jobs, webhooks y cron. El trabajo asíncrono no vive en el proceso HTTP.
   - `cmd/migrate`: herramienta de un solo uso que aplica las migraciones SQL embebidas (`goose`) con `up`, `down`, `status` y `version`. Corre antes de desplegar y termina.
3. **Estructura hexagonal liviana por módulo:** `domain/` → `app/{commands,queries}/` → `ports.go` → `adapters/{postgres,http}/`.
4. **Reglas de dependencia verificadas por herramienta, no por disciplina:**
   1. `domain` no importa infraestructura (ni nada del proyecto salvo `domain` compartidos mínimos).
   2. `app` depende de `domain` y de interfaces, nunca de `adapters`.
   3. Un módulo no importa internals de otro: interfaces expuestas o eventos de dominio.
   4. Las interfaces se definen del lado del consumidor, no una por struct.
   5. Todo servicio externo (Mercado Pago, geocoding, storage, email) está detrás de una interfaz con un fake para tests.

   El punto que hoy tiene dientes es el de dependencias externas: `depguard` (en `backend/.golangci.yml`) exige aprobar explícitamente cada librería de terceros. Las reglas de capas se suman en la Fase 1, cuando existan los módulos que separan.
5. **CQRS ligero, sin sobreingeniería:** comandos y queries separados; las queries pueden tener su propio repositorio de lectura con joins directos. Sin event sourcing ni bus externo.
6. **La caché es un decorador de la query**, no parte del dominio: `CachedBusinessSearch` envuelve a `BusinessSearch` en la Fase 7, sin tocar el dominio.
7. **Los webhooks se persisten primero** en una tabla y los procesa el worker, idempotentes y con reintentos. Nunca una goroutine suelta: se pierde trabajo al reiniciar.
8. **Stateless:** sin sesiones en memoria. JWT de acceso corto (≈15 min) más refresh rotativo guardado hasheado en la base. Solo dueños de local y super admin tienen cuenta; el cliente final usa un enlace con token no adivinable.
9. **12-factor:** configuración por variables de entorno, logs estructurados a stdout (`log/slog` con `request_id`), apagado ordenado.
10. **Postergado de forma explícita:** Redis (Fase 7, solo si el p95 de búsqueda lo justifica), CDN para imágenes, réplica de lectura y balanceador. Se agregan cuando haya métricas que lo pidan, no por las dudas.

## Consecuencias

- Un solo despliegue y una sola base simplifican operación, backups y transacciones: se puede escribir el canje de un código con `UPDATE ... WHERE status='pending' ... RETURNING` en vez de coordinar servicios.
- Las fronteras entre módulos son verificables en CI, así que la extracción futura de un módulo sigue siendo posible.
- El costo es disciplina estructural: la estructura de carpetas deja de ser un detalle y pasa a ser parte del contrato, con el andamiaje de `ports.go` y adaptadores por módulo.
- Sin Redis desde el arranque, la búsqueda p95 < 300 ms depende del índice GIST y de SQL bien escrito: es una restricción de diseño, no un accidente.

## Alternativas descartadas

- **Microservicios desde el inicio:** multiplica despliegues, consistencia distribuida y observabilidad para un dominio que todavía cambia. Se descarta hasta que haya una razón medida.
- **Un único binario para todo:** el trabajo asíncrono (webhooks, cron, emails) compite con las requests y no se puede escalar ni reiniciar por separado.
- **Arquitectura limpia con capas ceremoniales** (casos de uso, DTOs y mapeos por capa): más código de traducción que reglas de negocio, para un equipo de un par de personas.
- **Redis cache-aside desde el MVP:** agrega un servicio con estado que hay que operar, respaldar y monitorear antes de tener la métrica que lo justifique.
