# Software Design Document (SDD) — Plataforma MOVO  

> Versión 2 · Estado: borrador para revisión
> Principio rector: **primero el lazo de valor (buscar → obtener código → canjear), después el cobro, después la escala.**

---

## 1. Visión y alcance

### 1.1 Descripción
Plataforma SaaS (B2B2C) que conecta usuarios con actividades físicas y de bienestar (gimnasios, pilates, yoga, entrenadores, estética, piletas). Los establecimientos pagan una suscripción mensual (Mercado Pago) para aparecer en un mapa y ofrecer beneficios mediante códigos de descuento canjeables en el local.

### 1.2 Actores
| Actor | Qué hace |
|---|---|
| **Cliente (sin cuenta)** | Busca por cercanía/categoría; al generar un código o reservar deja sus datos de contacto (nombre + WhatsApp/email), sin registro ni verificación |
| **Dueño/Admin de local** | Paga suscripción, completa perfil, administra sedes, define beneficios y cupos, confirma reservas, canjea códigos |
| **Super Admin** | Aprueba/pausa/elimina locales, ve métricas, resuelve incidencias de cobro |

### 1.3 Objetivos del MVP
1. Un usuario encuentra un local cercano y obtiene un código en menos de 3 clics.
2. Un local canjea ese código en segundos desde el celular.
3. Un local solo es visible si tiene suscripción vigente (o piloto activo).
4. Desplegable con `docker compose up` y con CI verde.

### 1.4 Fuera de alcance del MVP (explícito)
- Agenda con cupos/capacidad por horario. La reserva del MVP es una **solicitud** (día/hora deseada) que el local confirma o rechaza manualmente.
- Reseñas y puntuaciones.
- App móvil nativa.
- Pagos del usuario final (solo se cobra al local).
- Réplicas de lectura, multi-región, Kubernetes.
- Notificaciones push.
- Seguimiento de ubicación en vivo del profesional (pendiente de definición, ver 12.2).

> **Decisión tomada:** el producto incluye **código de beneficio y reserva**, pero por capas: primero códigos (Fase 4), después reserva ligera (Fase 4B). La agenda con cupos se evalúa recién con evidencia de uso. Reseñas/puntuación quedan fuera del MVP.

### 1.5 Métricas de éxito (piloto)
- 5–10 locales de Almafuerte/Río Tercero activos.
- ≥ 50 códigos generados y ≥ 30 % canjeados en el primer mes.
- p95 de búsqueda < 300 ms.
- 0 códigos canjeados dos veces.

---

## 2. Stack tecnológico

| Capa | Elección | Justificación / nota |
|---|---|---|
| Backend | Go 1.22+ (`net/http` + `chi`) | Idiomático, poco acoplamiento al framework |
| Acceso a datos | `pgx` + `sqlc` | SQL explícito, tipado, sin magia de ORM |
| Migraciones | `goose` (SQL puro) | Versionadas en el repo, up/down en un archivo |
| DB | PostgreSQL 16 + **PostGIS** | Búsqueda por cercanía con índice espacial |
| Jobs | Cola sobre PostgreSQL (`river` o tabla propia con `FOR UPDATE SKIP LOCKED`) | No se pierden tareas al reiniciar |
| Caché | Redis (**Fase 7**, cuando haya métricas) | Cache-aside con TTL; también rate limiting |
| Frontend | Next.js (App Router) + Tailwind | SSR para SEO de fichas de locales |
| Mapas | React Leaflet + tiles configurables por variable de entorno | Piloto: tiles públicos de OSM; producción: Protomaps (PMTiles) o proveedor (sección 12) |
| Storage | S3 compatible: RustFS en local, gestionado (R2/B2/S3) en prod | URLs prefirmadas |
| Infra | Docker + docker compose; VPS al inicio | Simple y barato |
| CI | GitHub Actions | lint, tests, build, chequeo de migraciones |

> **Storage:** el código no habla con un servidor de objetos concreto, habla con el puerto `Storage` sobre la **API S3**. El servidor es un detalle de entorno: en desarrollo corre un S3 local (RustFS, con la imagen pineada), en producción un servicio gestionado (Cloudflare R2, Backblaze B2 o AWS S3). Cambiar de servidor es cambiar configuración, no código.

> Si preferís Gin (como en otros proyectos) es válido; lo importante es que los handlers HTTP sean una capa fina y reemplazable.

---

## 3. Arquitectura

### 3.1 Estilo: monolito modular (no microservicios)
Un solo repositorio, un solo despliegue, con **módulos con fronteras estrictas**. Se puede extraer un módulo a servicio más adelante si hay una razón real.

Dos binarios de servicio, misma base de código:
- `cmd/api`: servidor HTTP stateless (JWT), escalable detrás de un balanceador.
- `cmd/worker`: procesa jobs (webhooks, cron de reconciliación, emails).

Más `cmd/migrate`: herramienta de un solo uso que aplica las migraciones SQL embebidas (`goose`) con `up`, `down`, `status` y `version`. Se ejecuta antes de desplegar, no queda corriendo.

### 3.2 Estructura del repositorio
```
movo/
├── backend/
│   ├── go.mod
│   ├── go.sum
│   ├── cmd/
│   │   ├── api/main.go
│   │   ├── worker/main.go
│   │   └── migrate/main.go
│   ├── internal/
│   │   ├── platform/          # config, logger, db, httpx, jobs, clock, id
│   │   ├── identity/          # users, auth, roles
│   │   ├── business/          # locales, categorías, horarios, imágenes
│   │   ├── search/            # búsqueda geoespacial (solo lectura)
│   │   ├── benefits/          # beneficios y códigos
│   │   ├── billing/           # suscripciones + Mercado Pago
│   │   └── admin/             # casos de uso de super admin
│   ├── migrations/
│   └── sqlc.yaml
├── frontend/                  # Next.js
├── deploy/
│   ├── docker-compose.yml
│   └── docker-compose.prod.yml
├── docs/
│   ├── sdd.md
│   └── adr/                   # decisiones de arquitectura (1 archivo c/u)
├── CLAUDE.md
└── Makefile
```

### 3.3 Estructura interna de cada módulo (hexagonal liviana)
```
internal/benefits/
├── domain/            # entidades, value objects, errores, reglas (sin imports de infra)
├── app/
│   ├── commands/      # GenerateCode, RedeemCode  (escrituras)
│   └── queries/       # ListMyCodes               (lecturas)
├── ports.go           # interfaces que el módulo NECESITA (repos, clock, otros módulos)
└── adapters/
    ├── postgres/      # implementación con sqlc/pgx
    └── http/          # handlers, DTOs, validación
```

Reglas de dependencia (se validan con `depguard`/`go-arch-lint` en CI):
1. `domain` no importa nada del proyecto salvo otros `domain` compartidos mínimos.
2. `app` depende de `domain` y de **interfaces** (`ports.go`), nunca de `adapters`.
3. Un módulo **no importa internals de otro**; se comunica por interfaces expuestas o eventos de dominio.
4. Las interfaces se definen **del lado del consumidor** (idiom de Go), no "una interfaz por struct".
5. Servicios externos (Mercado Pago, geocoding, storage, email) siempre detrás de una interfaz con un fake para tests.

### 3.4 CQRS ligero (sin sobreingeniería)
- Comandos y queries separados en `app/commands` y `app/queries`.
- Las queries pueden usar un repositorio de lectura propio (SQL con joins directos).
- **La caché es un decorador** de la interfaz de query (`CachedBusinessSearch` envuelve a `BusinessSearch`). Se agrega en Fase 7 sin tocar el dominio.
- No hay event sourcing ni bus de mensajes externo.

### 3.5 Concurrencia y trabajos asíncronos
- Los webhooks **no** se procesan en una goroutine suelta: se guardan en una tabla y los toma el worker. Una goroutine en memoria pierde trabajo al reiniciar.
- Worker pool dentro de `cmd/worker` con concurrencia configurable, reintentos con backoff e idempotencia.
- `context.Context` en toda la cadena, con timeouts y cancelación en el apagado (`graceful shutdown`).

### 3.6 Stateless
- Sin sesiones en memoria. Auth con JWT de acceso corto (≈15 min) + refresh token rotativo guardado (hasheado) en DB, **solo para dueños/admins de local y super admin**. Los clientes no tienen cuenta: acceden a sus códigos/reservas con un enlace con token no adivinable (ver 5.2).
- Configuración por variables de entorno (12-factor). Logs a stdout.

### 3.7 Escalabilidad: qué se hace ahora y qué se posterga
| Elemento | Ahora | Cuándo agregarlo |
|---|---|---|
| Índice GIST en PostGIS | ✅ MVP | — |
| Servidor stateless | ✅ MVP | — |
| Redis cache-aside | ❌ | Fase 7, si p95 de búsqueda lo justifica |
| CDN para imágenes | ❌ (servir desde el bucket) | Fase 7 |
| Réplica de lectura | ❌ | Solo si CPU/IO de lecturas lo requiere (hay métricas) |
| Balanceador / múltiples instancias | ❌ (1 instancia) | Cuando haya tráfico real |

> Un local en Almafuerte/Río Tercero no necesita réplicas ni CDN. Diseñar *preparado* para escalar (stateless, interfaces) alcanza; implementarlo antes es costo sin retorno.

---

## 4. Modelo de datos (alto nivel)

Todas las tablas con `id` (UUID v7 o `bigserial`), `created_at`, `updated_at`. Migraciones en SQL puro.

| Tabla | Campos clave |
|---|---|
| `users` | email (único), password_hash (argon2id), role (`business_admin`/`super_admin`) — solo personal de locales y admins |
| `customers` | name, phone (normalizado), email (opcional), created_at — contacto declarado, **sin verificación ni contraseña** |
| `refresh_tokens` | user_id, token_hash, expires_at, revoked_at, replaced_by |
| `categories` | slug, name |
| `businesses` | owner_user_id (único: un dueño, un negocio), verified_at, category_id, name, slug (único), description, status (`draft`/`pending_review`/`active`/`paused`/`deleted`), is_visible, pilot |
| `branches` | business_id, name, address, city, `location geography(Point,4326)`, active — **la búsqueda se hace por sede** |
| `business_hours` | branch_id, weekday, opens_at, closes_at |
| `business_images` | business_id, object_key, position |
| `plans` | code (`starter`/`pro`), name, price_amount (ARS), currency, interval, mp_plan_id — **solo precio y vínculo con Mercado Pago**; los límites del plan salen de configuración (ver 4.2 y 7.5) |
| `services` | business_id, branch_id (opcional), name, description, active — lo que el negocio ofrece (clase de yoga, entrenamiento personal, etc.) |
| `service_schedules` | service_id, weekday, starts_at, ends_at |
| `verifications` | business_id, status (`pending`/`approved`/`rejected`), document_key (foto del DNI en storage privado, se elimina al resolverse), reviewed_by, reviewed_at, rejection_reason — define el sello "perfil verificado" |
| `business_events` | business_id, branch_id, type (`profile_view`/`whatsapp_click`/`code_generated`/`code_redeemed`/`reservation_requested`), occurred_at, source — **append-only y sin datos personales** |
| `subscriptions` | business_id, plan_id, mp_preapproval_id, status (`pending`/`authorized`/`paused`/`cancelled`), current_period_end, grace_until |
| `mp_webhook_events` | mp_event_id (único), topic, payload (jsonb), status (`received`/`processed`/`failed`), attempts, received_at |
| `benefits` | business_id, type (`percent`/`welcome_gift`), percent (ej. 10/15/20), gift_description, active, monthly_limit (configurable por el dueño) |
| `reservations` | branch_id, customer_id, requested_at (día/hora deseada), status (`requested`/`confirmed`/`rejected`/`cancelled`/`attended`), benefit_code_id (opcional), note |
| `benefit_codes` | code (único), benefit_id, business_id, customer_id, access_token_hash, status (`pending`/`redeemed`/`expired`), expires_at (= creación + 48 h), redeemed_at, redeemed_branch_id, redeemed_by |
| `jobs` | (si no se usa `river`) kind, payload, run_at, attempts, locked_at |
| `audit_log` | actor_id, action, entity, entity_id, metadata, created_at |

Índices importantes:
- `GIST (location)` en `branches`.
- `UNIQUE (code)` en `benefit_codes`; índice por `(customer_id, status)` y `(business_id, status)`.
- Índice único parcial para la regla *un código pendiente por cliente y local*: `UNIQUE (customer_id, business_id) WHERE status = 'pending'`.
- `UNIQUE (mp_event_id)` en `mp_webhook_events` (idempotencia).

### 4.1 Regla de visibilidad
Un local aparece en la búsqueda pública si y solo si:
```
business.status = 'active'
AND (subscription.status = 'authorized' OR now() <= subscription.grace_until OR business.pilot = true)
```
El campo `is_visible` se recalcula cuando cambia el estado del local o de la suscripción (evento de dominio) y la búsqueda filtra por él.

### 4.2 Planes y entitlements

Cualquier negocio (comercio o profesional) puede contratar cualquier plan; el tipo de negocio no restringe la elección.

| Característica | Starter ($19.800) | Pro ($29.800) |
|---|---|---|
| Perfil verificado (DNI) | ✅ | ✅ |
| Servicios activos | hasta 3 | ilimitados |
| Horarios por servicio | según servicios | ilimitados |
| Sedes | 2 | 4 |
| Link directo a WhatsApp con el alumno | ✅ | ✅ |
| Ubicación en el mapa (pin fijo) | ✅ | ✅ |
| Cupones de descuento (10/15/20 % o regalo) | ✅ | ✅ |
| Visibilidad en la plataforma | estándar | **destacada** |
| Métricas | básicas | **avanzadas de conversión** |
| Soporte | estándar | prioritario |

Reglas de implementación:
- **Límites desde configuración.** Los límites (servicios, sedes) se leen de variables de entorno a través de un puerto `PlanCatalog` (hoy `ConfigPlanCatalog`; mañana podría ser un `DBPlanCatalog` editable desde el panel sin tocar el dominio). El dominio consume un value object `Entitlements` (ej. `ent.CanAddBranch(current)`); **nunca** se pregunta `if plan == "pro"`.
- Los límites se validan al **crear o activar**, no al listar.
- **Downgrade o baja de límites** (Pro → Starter, o cambio de variable): no se borra nada. Se mantienen activos los elementos más antiguos hasta el nuevo límite y el resto pasa a `inactive`, con aviso al dueño.
- **Visibilidad destacada (Pro):** sello "Destacado" claramente etiquetado, prioridad dentro del mismo tramo de distancia y marcador distintivo en el mapa. El orden de búsqueda es por tramo de distancia (ej. cada 2 km), luego destacado, luego distancia exacta; así un Pro lejano no le gana a un local muy cercano.
- **Perfil verificado:** el dueño sube una foto de su DNI; el super admin la revisa y aprueba o rechaza. Minimización de datos: la imagen va a storage privado con acceso restringido y se **elimina al resolverse la verificación**; solo se conserva el resultado (`verified_at`, quién y cuándo). Requiere consentimiento expreso y política de privacidad (Ley 25.326). El sello se puede revocar.
- **Link a WhatsApp:** no se expone el número directamente; el botón pasa por `GET /r/wa/{branch_id}`, que registra un `whatsapp_click` y redirige a `https://wa.me/<número>?text=<mensaje prearmado>`.
- **Soporte prioritario:** es principalmente un proceso (canal y tiempos de respuesta definidos); en el sistema se refleja con una marca de prioridad en la bandeja del admin.
- **Métricas:**
  - *Básicas (todos los planes):* vistas del perfil, clics a WhatsApp, códigos generados y canjeados.
  - *Avanzadas (Pro):* embudo de conversión (vista → clic/código → canje), comparación por período, por beneficio/servicio y por sede.
  - Los eventos se registran **desde la Fase 3** para tener historial cuando se construya el panel; no se puede reconstruir lo que no se midió.

---

## 5. Flujos principales

### 5.1 Búsqueda por cercanía
1. El cliente (sin cuenta) envía `lat`, `lng`, `radius_m`, `category`, paginación (keyset o limit/offset acotado).
2. Query sobre `branches` con `ST_DWithin(location, point, radius)` ordenada por distancia (`<->`), uniendo con `businesses` y filtrando `is_visible = true`.
3. Respuesta con DTO liviano (sin campos internos).

### 5.2 Generación y canje de códigos
**Generar** (cliente sin cuenta)
1. El cliente elige un beneficio de un local visible y completa nombre + WhatsApp/email. Sin verificación.
2. Se valida: beneficio activo, **un solo código pendiente por cliente y local** (índice único parcial), y que el local no haya superado su `monthly_limit` del mes.
3. Código aleatorio criptográficamente seguro (`crypto/rand`), alfabeto sin caracteres ambiguos (sin `0/O/1/I`), 8 caracteres; si hay colisión por `UNIQUE`, se reintenta.
4. Se guarda `pending` con `expires_at = now() + 48 h` y se devuelve el código en pantalla junto con un enlace de acceso (token no adivinable, se guarda solo su hash) para volver a verlo.
5. **Anti-abuso (porque no hay verificación):** rate limit por IP y por dispositivo, captcha invisible (ej. Cloudflare Turnstile) en el formulario, normalización del teléfono/email para detectar duplicados y el cupo mensual del local como tope de daño.

**Canjear** (el `business_id` sale del token/permisos del admin, **nunca** del body)
```sql
UPDATE benefit_codes
SET status = 'redeemed', redeemed_at = now(), redeemed_by = $1
WHERE code = $2
  AND business_id = $3
  AND status = 'pending'
  AND expires_at > now()
RETURNING id;
```
Si no devuelve filas → inválido, vencido, ya canjeado o de otro local. Un solo `UPDATE` atómico evita doble canje por concurrencia.

Un job periódico marca como `expired` los vencidos (o se evalúa por `expires_at` en lectura).

### 5.2b Reserva ligera (Fase 4B)
1. El cliente elige sede, día y hora deseados y deja sus datos de contacto (mismas reglas anti-abuso).
2. Se crea la reserva en `requested` y se notifica al local (panel + email/WhatsApp según configuración).
3. El local la **confirma o rechaza** desde su panel; el cliente ve el estado en su enlace de acceso y se le avisa.
4. Una reserva confirmada puede incluir un beneficio (genera/asocia un código con la misma regla de 48 h, extendida hasta la fecha reservada si hace falta).
5. Sin control de cupos ni agenda: eso es un producto aparte y se decide con evidencia del piloto.

### 5.3 Suscripción con Mercado Pago
1. El dueño elige plan → el backend crea la suscripción (Preapproval) y devuelve la URL de checkout; se guarda `subscriptions.status = 'pending'`.
2. Mercado Pago redirige a `back_urls` → `/onboarding`. **Esa redirección NO confirma el pago**; es solo UX.
3. Mercado Pago llama al webhook:
   - Se valida la firma (`x-signature`) según la documentación oficial.
   - Se inserta el evento en `mp_webhook_events` (`ON CONFLICT DO NOTHING` por `mp_event_id`) y se responde `200` rápido.
4. El worker toma el evento, **consulta a la API de Mercado Pago el estado real** (no confía en el payload), actualiza `subscriptions` y recalcula `is_visible`.
5. Un cron diario de **reconciliación** compara el estado local con Mercado Pago por si se perdió algún webhook.
6. Pago rechazado/cancelado → **período de gracia de 7 días** (configurable) y aviso por email al dueño antes de ocultar el local. Confirmar en la documentación de Mercado Pago cuántos reintentos hace y cada cuánto, para alinear la gracia con ellos.

> Verificar siempre contra la documentación vigente de Mercado Pago (estados, topics y formato de firma pueden cambiar).

### 5.4 Panel Super Admin
- Alta, pausa, reactivación y baja lógica de locales (siempre con `audit_log`).
- Ver estado de suscripciones y eventos de webhook fallidos con opción de reintento.
- Marcar un local como "piloto" (visible sin suscripción).

---

## 6. API (contrato inicial, `/api/v1`)

| Método | Ruta | Rol | Descripción |
|---|---|---|---|
| POST | `/auth/register` · `/auth/login` · `/auth/refresh` · `/auth/logout` | público | Autenticación |
| GET | `/search/businesses` | público | Búsqueda geoespacial |
| GET | `/businesses/{slug}` | público | Ficha del local |
| POST/PUT | `/me/business` | business_admin | Crear/editar mi negocio |
| POST/PUT/DELETE | `/me/business/branches` | business_admin | Gestionar sedes (según límite del plan) |
| POST | `/me/business/images/presign` | business_admin | URL prefirmada de subida |
| POST/PUT | `/me/business/benefits` | business_admin | Gestionar beneficios |
| POST | `/benefits/{id}/codes` | público (captcha + rate limit) | Generar código con datos de contacto |
| GET | `/access/{token}` | token del cliente | Ver mis códigos y reservas |
| POST | `/branches/{id}/reservations` | público (captcha + rate limit) | Solicitar reserva |
| POST | `/me/business/reservations/{id}/confirm` · `/reject` | business_admin | Gestionar reservas |
| POST | `/me/business/codes/redeem` | business_admin | Canjear código |
| POST | `/billing/subscribe` | business_admin | Iniciar suscripción |
| POST | `/webhooks/mercadopago` | MP (firma) | Recibir eventos |
| * | `/admin/...` | super_admin | Gestión |

Convenciones: errores en el sobre JSON único de §7.2 (`code`, `message`, `details`), validación de entrada en la capa HTTP, paginación consistente, OpenAPI generado o mantenido en `docs/openapi.yaml`.

---

## 7. Requisitos no funcionales

### 7.1 Seguridad
- Contraseñas con argon2id (o bcrypt con costo adecuado); rate limiting en login y canje.
- JWT firmado con rotación de claves planificada; refresh tokens hasheados y rotativos.
- Autorización por recurso (un admin solo toca **su** local): se verifica `owner_user_id`/`business_id` en cada comando.
- Validación de uploads (tipo MIME real, tamaño máx.), URLs prefirmadas con expiración corta.
- Clientes sin cuenta: tokens de acceso aleatorios de ≥ 128 bits guardados hasheados, con expiración; captcha y rate limit en endpoints públicos que escriben.
- CORS restrictivo, cabeceras de seguridad, secretos solo por variables de entorno, nada en el repo.
- Dependencias auditadas (`govulncheck`, `npm audit`) en CI.
- Tratamiento de datos personales conforme a la **Ley 25.326** (política de privacidad, mínimo dato necesario).

### 7.2 Observabilidad
- Logs estructurados (`log/slog`) con `request_id`.
- Endpoints `/healthz` (liveness) y `/readyz` (DB lista).
- Todas las respuestas de error usan el mismo sobre JSON: `{"error":{"code":"...","message":"...","details":...}}`, con un `code` estable para el frontend, `details` opcional para errores por campo y sin filtrar detalles internos (el detalle técnico va al log).
- Cada request lleva `X-Request-Id`: se reusa el del cliente si es válido y, si no, se genera uno nuevo; el id viaja en el log de la request.
- Métricas (Prometheus) y trazas (OpenTelemetry) en Fase 8 o antes si hace falta.

### 7.3 Rendimiento
- Búsqueda p95 < 300 ms con 10k locales de prueba.
- Backups diarios de PostgreSQL con **prueba de restauración**.

### 7.4 Calidad de código
- `golangci-lint` (incl. `errcheck`, `govet`, `staticcheck`, `depguard`).
- Errores siempre manejados y envueltos (`fmt.Errorf("...: %w", err)`); errores de dominio tipados.
- Sin variables globales mutables; dependencias inyectadas por constructor (`main.go` es la raíz de composición).

### 7.5 Configuración de planes (variables de entorno)
```
PLAN_STARTER_MAX_BRANCHES=2
PLAN_STARTER_MAX_ACTIVE_SERVICES=3
PLAN_PRO_MAX_BRANCHES=4
PLAN_PRO_MAX_ACTIVE_SERVICES=0        # 0 = ilimitado
SEARCH_DISTANCE_BUCKET_M=2000         # tramo de distancia para ordenar destacados
```
Todas las instancias de la API deben tener los mismos valores; cambiarlos requiere reiniciar. Se validan al arrancar (el proceso falla si faltan o son inválidos).

---

## 8. Estrategia de testing

| Nivel | Qué cubre | Herramientas |
|---|---|---|
| Unitarios | Reglas de dominio (generación de código, visibilidad, estados de suscripción) | `testing`, `testify` |
| Integración | Repositorios y queries reales (PostGIS incluido) | `testcontainers-go` |
| Contrato / fakes | Mercado Pago y storage con implementaciones fake y webhooks de ejemplo | interfaces + fakes |
| Concurrencia | Canje simultáneo del mismo código (debe canjear exactamente 1) | tests con `errgroup` |
| API | Endpoints con DB de test | `httptest` |
| Frontend | Hooks y componentes clave | Vitest + Testing Library |
| E2E (mínimo) | Buscar → generar código → canjear | Playwright |

Meta de cobertura: ≥ 70 % global y ≥ 90 % en `domain` y `billing`.

---

## 9. Plan por fases

Convenciones de trabajo:
- **Una tarea = un commit pequeño y revisable** (idealmente < 400 líneas).
- Cada fase termina con una *Definition of Done* verificable y un tag de git.
- Tamaños: **S** (≤ 0,5 día) · **M** (≈ 1 día) · **L** (2–3 días).
- Flujo por tarea: leer SDD/CLAUDE.md → escribir test → implementar → lint+tests → commit.

---

### Fase 0 — Fundaciones
**Objetivo:** repo listo para trabajar con calidad desde el primer commit.

- [x] 0.1 Monorepo, `Makefile`, `.editorconfig`, `.gitignore` (S)
- [x] 0.2 `docker-compose` con Postgres+PostGIS y object storage S3 (RustFS) (S)
- [x] 0.3 `internal/platform`: config por env, `slog`, apagado ordenado (M)
- [x] 0.4 Conexión DB con `pgx` + migraciones + primera migración (extensión PostGIS) (S)
- [x] 0.5 Servidor HTTP con `chi`, `request_id`, `/healthz`, `/readyz`, formato de error (M)
- [x] 0.6 CI (lint + test + build + chequeo de migraciones) (M)
- [x] 0.7 `docs/adr/0001` con las decisiones de la sección 3 y `docs/adr/0002` de object storage (puerto `Storage` con API S3; RustFS pineado **sólo en desarrollo**, Garage como plan B, servicio gestionado en producción), y `CLAUDE.md` actualizado (S)
- [ ] 0.8 Frontend Next.js + Tailwind base, ESLint/Prettier, layout (S)

**DoD:** `make up` levanta todo; CI verde; `/readyz` responde con la DB conectada.

---

### Fase 1 — Identidad y autorización (dueños y admins)
**Objetivo:** login seguro para dueños de local y super admin. Los clientes no tienen cuenta.

- [ ] 1.1 Migraciones `users`, `refresh_tokens` (S)
- [ ] 1.2 Dominio `identity`: entidad User, validaciones, hashing (M)
- [ ] 1.3 Casos de uso Register / Login / Refresh / Logout (M)
- [ ] 1.4 Middleware JWT + middleware de roles (S)
- [ ] 1.5 Rate limiting en login (en memoria por ahora, interfaz lista para Redis) (S)
- [ ] 1.6 Frontend: páginas login/registro de dueños y manejo de sesión en hooks (M)

**DoD:** tests unitarios + integración de auth; un dueño no accede a rutas de super admin ni a datos de otro negocio.

---

### Fase 2 — Negocios y onboarding (sin cobro todavía)
**Objetivo:** que un local pueda existir y ser aprobado manualmente.

- [ ] 2.1 Migraciones `categories`, `businesses`, `branches`, `business_hours`, `business_images` (con PostGIS) (S)
- [ ] 2.2 Dominio `business` (estados, slug, validaciones) (M)
- [ ] 2.3 Casos de uso: crear/editar mi negocio, agregar/editar sedes y horarios (un dueño = un negocio) (M)
- [ ] 2.4 Puerto `Storage` + adaptador S3 (RustFS en local, R2/B2/S3 en prod) con URL prefirmada y validación; endpoint, región y modo path-style configurables; test de integración (tag `integration`) que sube y baja un objeto con URL prefirmada contra el servidor local (M)
- [ ] 2.5 Super admin: aprobar / pausar / reactivar / baja lógica + `audit_log` (M)
- [ ] 2.5b Migraciones y dominio de `services` y `service_schedules` (CRUD, activar/desactivar) (M)
- [ ] 2.5c Flujo de verificación: subida de la foto del DNI a storage privado, aprobación/rechazo del super admin con sello `verified_at` y eliminación de la imagen al resolverse (M)
- [ ] 2.6 Frontend onboarding: formulario, subida de fotos, selector de ubicación en mapa (L)

**DoD:** un dueño crea su local, el admin lo activa manualmente y queda en estado `active`.

---

### Fase 3 — Búsqueda geoespacial y mapa
**Objetivo:** el usuario encuentra locales cercanos.

- [ ] 3.1 Query sobre `branches` con `ST_DWithin`, orden por tramo de distancia → destacado → distancia exacta, y filtros por categoría (M)
- [ ] 3.2 Endpoint de búsqueda con paginación y DTO liviano (S)
- [ ] 3.3 Seed de 10k locales falsos y medición p95 con `EXPLAIN ANALYZE` (S)
- [ ] 3.4 Frontend: mapa con React Leaflet, marcadores, lista sincronizada, geolocalización del navegador (L)
- [ ] 3.5 Ficha SSR del local `/locales/[slug]` con metadatos SEO, servicios, horarios y sello de verificado (M)
- [ ] 3.5b Endpoint `GET /r/wa/{branch_id}` que registra el clic y redirige a `wa.me` con mensaje prearmado (S)
- [ ] 3.5c Registro de eventos en `business_events` (vistas de perfil, clics) sin datos personales y con filtrado básico de bots (M)
- [ ] 3.6 Puerto `Geocoder` con fake, adaptador Nominatim con rate limiter (1 req/s) + caché, y User-Agent identificatorio; tiles configurables por env (M)

**DoD:** búsqueda p95 < 300 ms; la ficha se renderiza del lado del servidor.

---

### Fase 4 — Beneficios y códigos (el lazo de valor)
**Objetivo:** generar y canjear códigos de forma segura y atómica, sin cuenta de cliente.

- [ ] 4.1 Migraciones `customers`, `benefits`, `benefit_codes` + índice único parcial (S)
- [ ] 4.2 Dominio `benefits`: tipos (`percent` 10/15/20 o `welcome_gift`), TTL de 48 h, cupo mensual, estados (M)
- [ ] 4.3 CRUD de beneficios y cupo mensual desde el panel del local (M)
- [ ] 4.4 Comando `GenerateCode`: datos de contacto, un pendiente por cliente y local, cupo, reintento por colisión, token de acceso (M)
- [ ] 4.5 Anti-abuso: rate limit por IP/dispositivo + captcha (Turnstile) tras una interfaz `CaptchaVerifier` (M)
- [ ] 4.6 Comando `RedeemCode` atómico + test de concurrencia (M)
- [ ] 4.7 Query `GET /access/{token}` y panel del local con canjes recientes (S)
- [ ] 4.8 Job de expiración de códigos (S)
- [ ] 4.9 Frontend: "obtener código" (formulario liviano) y pantalla de canje rápida para celular (L)

**DoD:** test de concurrencia demuestra un único canje; el flujo completo funciona de punta a punta y los intentos de abuso quedan frenados por rate limit/captcha.

> **Hito de piloto:** con las Fases 0–4 podés cargar 5–10 locales reales marcados como `pilot` y **validar el producto antes de integrar pagos**.

---

### Fase 4B — Reserva ligera
**Objetivo:** el cliente solicita una visita y el local la confirma.

- [ ] 4B.1 Migración `reservations` (S)
- [ ] 4B.2 Dominio: estados y transiciones válidas (`requested` → `confirmed`/`rejected`/`cancelled`/`attended`) (M)
- [ ] 4B.3 Comando de solicitud con las mismas reglas anti-abuso (M)
- [ ] 4B.4 Confirmar/rechazar desde el panel del local (S)
- [ ] 4B.5 Puerto `Notifier` (email + enlace `wa.me`) con fake y job de envío (M)
- [ ] 4B.6 Frontend: formulario de reserva y vista de estado por enlace de acceso (L)

**DoD:** una solicitud recorre `requested` → `confirmed` con notificación; transiciones inválidas son rechazadas por el dominio.

---

### Fase 5 — Suscripciones con Mercado Pago
**Objetivo:** cobro recurrente y visibilidad automática.

- [ ] 5.1 Migraciones `plans` (Starter $19.800 / Pro $29.800 ARS), `subscriptions`, `mp_webhook_events` (S)
- [ ] 5.2 Puerto `PaymentProvider` + fake para tests (M)
- [ ] 5.3 Adaptador Mercado Pago: crear Preapproval y consultar estado (M)
- [ ] 5.4 Endpoint `/billing/subscribe` y vuelta a `/onboarding` (S)
- [ ] 5.5 Endpoint de webhook: validar firma, guardar evento, responder 200 (M)
- [ ] 5.6 Worker: procesar evento consultando la API de MP, idempotente, con reintentos (L)
- [ ] 5.7 Regla de visibilidad con período de gracia de 7 días + evento de dominio (M)
- [ ] 5.7b Proceso de actualización de precio de un plan existente (inflación) (M)
- [ ] 5.7c Puerto `PlanCatalog` + `ConfigPlanCatalog` (env), value object `Entitlements` y validación en creación/activación de servicios y sedes; política de downgrade sin borrar datos (M)
- [ ] 5.8 Cron de reconciliación diario (M)
- [ ] 5.9 Frontend: elegir plan, estado de suscripción y aviso de pago pendiente (M)

**DoD:** simulando eventos (aprobado, rechazado, cancelado, duplicado, fuera de orden) el estado y la visibilidad son siempre correctos.

---

### Fase 6 — Panel Super Admin completo
- [ ] 6.1 Listado/filtros de locales y suscripciones (M)
- [ ] 6.2 Vista y reintento de webhooks fallidos (S)
- [ ] 6.3 Métricas del super admin: locales activos, códigos generados/canjeados (M)
- [ ] 6.3b Panel de métricas del local: básicas para todos y embudo de conversión avanzado solo con `advanced_metrics` (L)
- [ ] 6.3c Marca de prioridad en la bandeja del admin para locales con `priority_support` (S)
- [ ] 6.4 Frontend admin (L)

**DoD:** el admin puede resolver un problema de cobro sin tocar la base de datos.

---

### Fase 7 — Rendimiento y escala (guiada por métricas)
- [ ] 7.1 Decorador de caché Redis (cache-aside, TTL, clave por geocelda/categoría) (M)
- [ ] 7.2 Invalidación al cambiar un local o su visibilidad (M)
- [ ] 7.3 Rate limiting distribuido con Redis (S)
- [ ] 7.4 CDN delante del bucket de imágenes y optimización de imágenes (M)
- [ ] 7.5 Prueba de carga (k6) y ajuste de pool de conexiones (M)
- [ ] 7.6 Evaluar réplica de lectura **solo si las métricas lo exigen** (ADR)

**DoD:** mejora medible y documentada contra la línea base de la Fase 3.

---

### Fase 8 — Hardening y lanzamiento
- [ ] 8.1 Checklist de seguridad (sección 7.1) y `govulncheck`/`npm audit` en CI (M)
- [ ] 8.2 Observabilidad: métricas y trazas (M)
- [ ] 8.3 Backups automáticos + prueba de restauración (M)
- [ ] 8.4 Despliegue en VPS con `docker-compose.prod.yml`, HTTPS y proxy reverso (L)
- [ ] 8.5 Términos y condiciones, política de privacidad, definición de facturación (S)
- [ ] 8.6 Runbook (qué hacer si cae el webhook, si falla MP, cómo restaurar) (S)

**DoD:** sistema en producción, monitoreado y con piloto activo.

---

## 10. Definition of Done global (por tarea)
- Compila, pasa lint y todos los tests.
- Tests nuevos para la lógica agregada.
- Sin secretos ni TODOs críticos.
- Migración reversible (up/down) cuando aplica.
- Documentación actualizada (README/ADR/OpenAPI) si cambia el comportamiento.

---

## 11. Riesgos y mitigaciones

| Riesgo | Mitigación |
|---|---|
| Servidor público de **Nominatim** (1 req/s por aplicación, sin autocompletado) y **tiles de OSM** (no pensados para producción a escala) | Puerto `Geocoder` y tiles configurables por env; geocodificar solo al dar de alta una sede, con caché; en producción usar Protomaps/proveedor (sección 12) |
| Webhooks perdidos o duplicados | Tabla de eventos idempotente + reconciliación diaria |
| Doble canje de código | `UPDATE` atómico + test de concurrencia |
| Abuso de códigos/reservas al no haber cuenta ni verificación | Rate limit, captcha, normalización de contacto, un pendiente por cliente/local y cupo mensual; si hay abuso real, agregar verificación por WhatsApp/email |
| Precios fijos en ARS con inflación | Precios en tabla `plans` y proceso definido para actualizar suscripciones existentes |
| Sobreingeniería temprana (réplicas, CDN, microservicios) | Fase 7 solo con métricas; ADRs para cada decisión |
| Los locales no ven valor y no pagan | Piloto gratuito antes de la Fase 5, medir canjes |
| Facturación/impuestos de la suscripción | Definir con un contador antes del lanzamiento (Fase 8) |
| Datos personales de usuarios | Mínimo dato necesario, política de privacidad, Ley 25.326 |
| Fotos de DNI de los dueños | Storage privado con acceso restringido, consentimiento expreso, eliminación al resolver la verificación y consulta legal sobre obligaciones de registro de bases de datos |

---

## 12. Decisiones

### 12.1 Decisiones tomadas
| # | Tema | Decisión |
|---|---|---|
| 1 | Producto | Código de beneficio **y** reserva, por capas (Fase 4 y 4B); reserva = solicitud confirmada por el local, sin agenda de cupos |
| 2 | Códigos | Un beneficio pendiente por local, válido 48 h; el dueño configura el límite mensual |
| 3 | Beneficios | Porcentaje (10/15/20 %) o regalo de bienvenida, elegido por el dueño |
| 4 | Cantidad | Un local puede tener todos los beneficios que quiera |
| 5 | Cuenta del cliente | No hace falta para ver locales; se piden datos al generar código o reservar |
| 6 | Verificación | Sin verificación de email (se compensa con anti-abuso) |
| 7 | Planes | Starter (entrenadores y profesionales) $19.800 · Pro $29.800 |
| 8 | Sedes | Un dueño = un negocio, con múltiples sedes |
| 9 | Gracia | 7 días ante pago rechazado |
| 10 | Mapas y geocoding | Ver 12.3 |
| 11 | Ubicación | Pin fijo en el mapa; seguimiento en vivo queda fuera |
| 12 | Cupones | Ambos planes ofrecen beneficios estándar (10/15/20 % o regalo); Pro se diferencia por visibilidad destacada |
| 13 | Elegibilidad de planes | Cualquier negocio puede contratar Starter o Pro |
| 14 | Sedes | Starter 2, Pro 4, configurables por variable de entorno |
| 15 | Verificación | Se pide solo el DNI; imagen eliminada al resolver |

### 12.2 Pendientes de definir
- **Sedes de Starter:** se interpretó "una sede más" como 2 sedes en total (principal + 1). Confirmar.
- **Cupo mensual:** ¿es un tope de códigos por local por mes (asumido) o por cliente?
- **Datos que se piden al cliente:** ¿WhatsApp obligatorio? (recomendado como contacto principal; email opcional)
- **Mes de prueba o piloto gratuito:** ¿cuánto dura y qué locales entran?
- **Beneficios y sedes:** ¿un beneficio vale en todas las sedes del negocio? (se asume que sí)
- **Visibilidad destacada:** confirmar el alcance propuesto (sello "Destacado", prioridad dentro del mismo tramo de distancia y marcador distintivo).

### 12.3 Mapas y geocodificación (decisión recomendada)
| Necesidad | Piloto (Fases 2–4) | Producción |
|---|---|---|
| Tiles del mapa | Tiles públicos de OSM con URL en variable de entorno, atribución y caché | Protomaps (archivo PMTiles propio en el bucket) o proveedor como MapTiler |
| Geocodificar la dirección de una sede | Nominatim público, solo al dar de alta una sede, con User-Agent propio, caché y limitador global de 1 req/s | Mismo adaptador, o Nominatim/Photon autoalojado si el volumen crece |
| Ubicación del cliente | `navigator.geolocation` del navegador (sin geocoding) | Igual |
| Buscar "mi ciudad/barrio" | Lista local de localidades cargada en DB | Igual, o API Georef del Estado argentino si sigue activa |
| Autocompletado de direcciones | No (la política de Nominatim público lo prohíbe) | Photon autoalojado, solo si se justifica |

---

## 13. Glosario
- **B2B2C:** el negocio paga, el consumidor final usa.
- **Preapproval:** suscripción recurrente de Mercado Pago.
- **Idempotencia:** procesar el mismo evento dos veces produce el mismo resultado.
- **Cache-aside:** la app consulta la caché, y si falla, lee de la DB y completa la caché.
- **ADR:** Architecture Decision Record, documento corto que registra una decisión y su motivo.