# Contexto del Proyecto: Plataforma MOVO

## Visión General
Plataforma SaaS (B2B2C) que conecta usuarios con actividades físicas y de bienestar (gimnasios, pilates, yoga, etc.). Los locales pagan una suscripción mensual (Mercado Pago) para aparecer en un mapa y ofrecer beneficios mediante códigos canjeables en el local.

Fuente de verdad: `docs/sdd.md` (fases, modelo de datos, flujos). Decisiones de arquitectura: `docs/adr/`.

## Stack
- **Backend:** Go 1.27+ (el módulo declara `go 1.27.0`), `chi`, `pgx` + `sqlc`, PostgreSQL + PostGIS, migraciones SQL puras (`goose`).
- **Jobs:** cola sobre PostgreSQL (`river` o `FOR UPDATE SKIP LOCKED`). Redis solo desde la Fase 7.
- **Frontend:** Next.js (App Router) + Tailwind. Mapas: OpenStreetMap + React Leaflet.
- **Infra:** Docker / docker compose, entorno enfocado en la terminal (`make`).
- **Storage:** puerto `Storage` con API S3 (URLs prefirmadas). En local, RustFS pineado en `docker compose`; en producción, servicio gestionado (R2, B2 o S3). El código nunca conoce un servidor de objetos concreto y el servidor local es sólo de desarrollo.

## Arquitectura (monolito modular)
- Dos binarios de servicio en `backend/`: `cmd/api` (HTTP stateless con JWT) y `cmd/worker` (jobs, webhooks, cron). Más `cmd/migrate`: herramienta de un solo uso que aplica las migraciones SQL embebidas (`up`/`down`/`status`/`version`) antes de desplegar.
- Módulos en `backend/internal/`: `identity`, `business`, `search`, `benefits`, `billing`, `admin`, `platform`.
- Cada módulo: `domain/` → `app/{commands,queries}/` → `ports.go` → `adapters/{postgres,http}/`.
- Reglas de dependencia:
  1. `domain` no importa infraestructura.
  2. `app` depende de interfaces, nunca de `adapters`.
  3. Un módulo no importa internals de otro; usa interfaces o eventos de dominio.
  4. Las interfaces se definen del lado del consumidor.
  5. Todo servicio externo (Mercado Pago, geocoding, storage, email) va detrás de una interfaz con fake para tests.
- CQRS ligero: comandos y queries separados; la caché (Redis) es un **decorador** de la query, no parte del dominio.
- Contrato HTTP (`internal/platform/httpx`): sobre de error único `{"error":{"code","message","details"}}` con `code` estable, `X-Request-Id` por request, y `/healthz` (liveness) + `/readyz` (base lista) sin filtrar detalles internos al cliente.
- Composición de dependencias solo en `main.go`; nada de variables globales mutables.

## Reglas de Desarrollo (Go)
- Código idiomático; errores siempre manejados y envueltos con `%w`; errores de dominio tipados.
- `context.Context` en toda la cadena; apagado ordenado.
- Los webhooks se **persisten primero** y se procesan en el worker (idempotentes, con reintentos). No usar goroutines sueltas para trabajo que no se puede perder.
- Canje de códigos con `UPDATE ... WHERE status='pending' ... RETURNING` (atómico). El `business_id` sale del token, nunca del body.
- Validar la firma de los webhooks y **consultar a Mercado Pago el estado real**; la redirección `back_urls` no confirma pagos.
- Migraciones SQL puras con up/down. Logs con `slog`. Config por variables de entorno.

## Reglas de Desarrollo (Frontend)
- Componentes limpios; lógica de negocio y llamadas a la API en hooks personalizados.
- SSR para páginas públicas (fichas de locales); mapas solo en componentes cliente.
- Tipos compartidos desde el contrato OpenAPI cuando sea posible.

## Flujo de trabajo (IMPORTANTE)
- Ramas: el trabajo sale de `develop` y los PR van contra `develop`. `main` solo recibe merge cuando se publica la v1.0.0; nunca se commitea directo ahí.
- Trabajar **una tarea del SDD por vez** (ej. "Tarea 4.4"). No adelantarse a otras fases.
- Cada tarea = un commit chico (< 400 líneas). Si crece, dividirla y avisar.
- Orden por tarea: leer SDD → escribir test → implementar → `make lint test build` → commit.
- No introducir Redis, CDN ni réplicas antes de la Fase 7 salvo que haya métricas que lo justifiquen (crear ADR).
- Si algo del SDD es ambiguo o contradice el código, preguntar antes de decidir.
- Los clientes finales NO tienen cuenta (datos de contacto + token de acceso); JWT solo para dueños y super admin.
- Los límites de plan (sedes, servicios) se leen de configuración por env a través del puerto `PlanCatalog`; nunca preguntar `if plan == "pro"`, usar `Entitlements`.
- Entregar código completo y listo para usar, no fragmentos parciales.

## Comandos útiles
- `make up` / `make down`: levantar/bajar servicios.
- `make migrate-up` / `make migrate-down` / `make migrate-status`: migraciones (leen `DATABASE_URL` del `.env`, o la del entorno si está definida).
- `make lint` / `make test` / `make build`: lo mismo que corre CI (`.github/workflows/ci.yml`: lint, tests con `-race`, build e ida y vuelta de migraciones).
- Frontend (`cd frontend`): `npm run dev`, `npm run lint`, `npm run typecheck`, `npm run format` / `npm run format:check`, `npm run build`. CI corre los cinco sobre Node 24.22.0, declarado en `frontend/.nvmrc`.
- En `frontend/package.json` las versiones van **exactas** (sin `^` ni `~`): actualizar es una tarea, no un efecto colateral de `npm install`.

## Testing
- Unitarios en `domain`; integración con `testcontainers-go` (incluye PostGIS); fakes para Mercado Pago y storage.
- Test de concurrencia obligatorio para el canje de códigos.
- Meta: ≥ 70 % global, ≥ 90 % en `domain` y `billing`.