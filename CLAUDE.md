# Contexto del Proyecto: Plataforma MOVO

## Visión General
Plataforma SaaS (B2B2C) que conecta usuarios con actividades físicas y de bienestar (gimnasios, pilates, yoga, etc.). Los locales pagan una suscripción mensual (Mercado Pago) para aparecer en un mapa y ofrecer beneficios mediante códigos canjeables en el local.

Fuente de verdad: `docs/sdd.md` (fases, modelo de datos, flujos). Decisiones de arquitectura: `docs/adr/`.

## Principios del MVP y Reglas de Negocio (SDD §1.2)
1. **Cero proveedores externos de mensajería o pagos:** No se integra la API de Mercado Pago, ni WhatsApp Business, ni email, ni push. El contacto con el local y las reservas se resuelven exclusivamente por enlace `wa.me`. Sin reservas con estados ni agenda con cupos en el MVP.
2. **Mercado Pago en el MVP:** Mercado Pago NO se integra por API en el MVP: solo links de planes de suscripción por env; el super admin confirma el alta y registra cada pago mensual.
3. **Contar, no seguir:** Las métricas salen de contadores agregados sin datos personales (sin IP, user-agent, identificadores ni recorridos individuales). Se rotulan como "vistas", no como "personas".
4. **El cliente no tiene cuenta:** Solo deja nombre y WhatsApp cuando genera un cupón (sin registro ni verificación). JWT solo para dueños y super admin.
5. **Vistas y backend se construyen juntos por fase** (desarrollo vertical), para ver el flujo de punta a punta.
6. **Glosario de interfaz:** se usa "cupón" en la interfaz y el marketing; el código alfanumérico es el "código del cupón"; en base de datos la tabla es `benefit_codes`.
7. **Entitlements / PlanCatalog:** Los límites de plan (sedes, servicios) se leen de configuración por env a través del puerto `PlanCatalog`; nunca preguntar `if plan == "pro"`, usar `Entitlements`.

## Stack
- **Backend:** Go 1.27+ (el módulo declara `go 1.27.0`), `chi`, `pgx` + `sqlc`, PostgreSQL + PostGIS, migraciones SQL puras (`goose`).
- **Background / Tareas:** tareas programadas en `cmd/worker` con lock consultivo de PostgreSQL. Redis solo desde la Fase 7.
- **Frontend:** Next.js (App Router) + Tailwind. Mapas: OpenStreetMap + React Leaflet.
- **Infra:** Docker / docker compose, entorno enfocado en la terminal (`make`).
- **Storage:** puerto `Storage` con API S3 (URLs prefirmadas). En local, RustFS pineado en `docker compose`; en producción, servicio gestionado (R2, B2 o S3). El código nunca conoce un servidor de objetos concreto y el servidor local es sólo de desarrollo.

## Arquitectura (monolito modular)
- Dos binarios de servicio en `backend/`: `cmd/api` (HTTP stateless con JWT) y `cmd/worker` (tareas programadas / cron). Más `cmd/migrate`: herramienta de un solo uso que aplica las migraciones SQL embebidas (`up`/`down`/`status`/`version`) antes de desplegar.
- Módulos en `backend/internal/`: `identity`, `business`, `search`, `benefits`, `billing`, `admin`, `platform`.
- Cada módulo: `domain/` → `app/{commands,queries}/` → `ports.go` → `adapters/{postgres,http}/`.
- Reglas de dependencia:
  1. `domain` no importa infraestructura.
  2. `app` depende de interfaces, nunca de `adapters`.
  3. Un módulo no importa internals de otro; usa interfaces o eventos de dominio.
  4. Las interfaces se definen del lado del consumidor.
  5. Los servicios externos reales (geocoding, storage) siempre van detrás de una interfaz con fake para tests. Mercado Pago NO se integra por API en el MVP.
- CQRS ligero: comandos y queries separados; la caché (Redis) es un **decorador** de la query, no parte del dominio.
- Contrato HTTP (`internal/platform/httpx`): sobre de error único `{"error":{"code","message","details"}}` con `code` estable, `X-Request-Id` por request, y `/healthz` (liveness) + `/readyz` (base lista) sin filtrar detalles internos al cliente.
- Composición de dependencias solo en `main.go`; nada de variables globales mutables.

## Reglas de Desarrollo (Go)
- Código idiomático; errores siempre manejados y envueltos con `%w`; errores de dominio tipados.
- `context.Context` en toda la cadena; apagado ordenado.
- Tareas programadas en el worker con lock consultivo en Postgres; no usar goroutines sueltas para trabajo que no se puede perder.
- Canje de códigos con `UPDATE ... WHERE status='pending' ... RETURNING` (atómico). El `business_id` sale del token, nunca del body.
- Migraciones SQL puras con up/down. Logs con `slog`. Config por variables de entorno.

## Reglas de Desarrollo (Frontend)
- Componentes limpios; lógica de negocio y llamadas a la API en hooks personalizados.
- SSR para páginas públicas (fichas de locales); mapas solo en componentes cliente.
- Tipos compartidos desde el contrato OpenAPI cuando sea posible.
- Los paths (rutas de URL y carpetas de ruta de `app/`) van en inglés (`/reviews`, no `/resenas`); el texto que ve el usuario sigue en español. Todos los paths del proyecto están en inglés (el área del dueño es `/owners`, las fichas `/businesses/[slug]`, los cupones `/coupon/[token]`): toda ruta nueva sigue esa regla.

## Vistas (frontend)
- Backend y vistas se desarrollan juntos por fase. Al empezar la fase, listame las vistas que necesita y los endpoints que consume, sin pedirme las imágenes todavía.
- Nunca inventes una vista: al llegar a su tarea, frená y pedime la imagen con su nombre exacto, avisando que el backend ya está listo.
- Con la imagen: implementala fiel al diseño, con los tokens de marca y los componentes compartidos, textos del SDD y contra la API real de la fase. Si la imagen contradice el SDD, avisame y seguí el SDD.
- Al terminar cada vista, decime cómo verla de punta a punta: comandos, URL, datos de prueba (creá `make seed-dev` en esa misma tarea si hace falta) y qué flujo probar.

## Flujo de trabajo (IMPORTANTE)
- Una fase del SDD por prompt (las fases 4 y 5 se pueden partir en dos mitades que indique el prompt).
- Leé solo CLAUDE.md, la sección de la fase en docs/sdd.md y las secciones de arquitectura o datos que esa fase cite. No releas el SDD completo.
- Ejecutá las tareas en orden. Por cada tarea: test primero si aplica, implementación, `make lint test`, un commit chico (~400 líneas como guía) en la rama de la fase y tildá la tarea en docs/sdd.md dentro del mismo commit. No pidas aprobación por commit.
- El estado es el SDD: la primera tarea sin tildar es por donde se sigue. Si la sesión se corta, retomá desde ahí.
- Frená y preguntame si: hay ambigüedad que afecta modelo de datos, seguridad o cobros; algo contradice el SDD; un test falla tras 2 intentos; hace falta una dependencia no prevista; o necesitás la imagen de una vista.
- Nunca pushear a main, nunca force push, nunca borrar ramas ni datos; push solo de la rama de la fase y solo si te lo pido.
- Al cerrar la fase: verificá la Definition of Done, corré `make ci` y dame un resumen corto con `git log --oneline` y `git diff --stat`, sin diffs completos.
- Respuestas breves y sin releer lo ya leído.

## Modo de trabajo (ODD)
- Entender antes de cambiar: explorá lo necesario y confirmá que entendiste la tarea del SDD. Si el pedido es solo una consulta, respondé sin modificar nada.
- Cambio chico y entendido: implementalo directo, sin planes ni documentos extra.
- Trabajo sustancial (seguridad, cobros, datos, varias tareas): antes de empezar, escribí una nota corta en docs/work/<tarea>.md con objetivo, decisiones, hecho y pendiente, y retomá desde ahí si se corta la sesión. El estado general sigue siendo el SDD.
- Verificá siempre: tests y lint antes de cerrar; después tildá la tarea y dejá un resumen corto.
- No generes specs ni diseños formales por cada cambio, salvo que lo pida.
- Para escribir código seguí el skill small-diffs: las ~400 líneas son una guía, no un tope; si un cambio las supera de forma natural, explicalo brevemente y seguí. Frená solo si hay una decisión de producto, seguridad o cobros.
- Los skills son guías generales: si uno contradice docs/sdd.md o este archivo, gana el proyecto. Para Next.js, verificá la versión en frontend/package.json antes de aplicar patrones del skill nextjs-15.

## Comandos útiles
- `make up` / `make down`: levantar/bajar servicios.
- `make migrate-up` / `make migrate-down` / `make migrate-status`: migraciones (leen `DATABASE_URL` del `.env`, o la del entorno si está definida).
- `make lint` / `make test` / `make build`: lo mismo que corre CI (`.github/workflows/ci.yml`: lint, tests con `-race`, build e ida y vuelta de migraciones).
- Frontend (`cd frontend`): `npm run dev`, `npm run lint`, `npm run typecheck`, `npm run format` / `npm run format:check`, `npm run build`. CI corre los cinco sobre Node 24.22.0, declarado en `frontend/.nvmrc`.
- En `frontend/package.json` las versiones van **exactas** (sin `^` ni `~`): actualizar es una tarea, no un efecto colateral de `npm install`.

## Testing
- Unitarios en `domain`; integración con `testcontainers-go` (incluye PostGIS); fake para storage.
- Test de concurrencia obligatorio para el canje de códigos.
- Meta: ≥ 70 % global, ≥ 90 % en `domain` y `billing`.