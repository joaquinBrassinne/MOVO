# Software Design Document (SDD) — Plataforma MOVO

> Versión 3 · Estado: alcance del MVP cerrado
> Principio rector: **primero el lazo de valor (buscar → obtener cupón → canjear), después el cobro manual, después la automatización y la escala.**

---

## 1. Visión y alcance

### 1.1 Descripción
Plataforma SaaS (B2B2C) que conecta personas con actividades físicas y de bienestar (gimnasios, pilates, yoga, entrenadores, estética, natación, clubes de pádel y guías y empresas de trekking y outdoor: trekking, cabalgatas, kayak, mountain bike y escalada). Los establecimientos pagan una suscripción mensual mediante un **plan de suscripción de Mercado Pago (cobro automático)** para aparecer en un mapa y ofrecer cupones de descuento canjeables en el local. Los clientes contactan al local por WhatsApp para reservar clases.

### 1.2 Principios del MVP
1. **Cero proveedores externos de mensajería o pagos.** No se integra la API de Mercado Pago, ni WhatsApp Business, ni email, ni push. El contacto con el local es un enlace `wa.me`.
2. **Contar, no seguir.** Las métricas salen de contadores agregados sin datos personales (sin IP, user-agent, identificadores ni recorridos individuales). Se rotulan como "vistas", no como "personas".
3. **El cliente no tiene cuenta.** Solo deja nombre y WhatsApp cuando genera un cupón.
4. **Vistas y backend se construyen juntos por fase** (desarrollo vertical), para ver el flujo de punta a punta.
5. **Glosario de interfaz:** se usa "cupón" en la interfaz y el marketing; el código alfanumérico es el "código del cupón"; en base de datos la tabla es `benefit_codes`.

### 1.3 Actores
| Actor | Qué hace |
|---|---|
| **Cliente (sin cuenta)** | Busca por cercanía/categoría, contacta al local por WhatsApp, genera cupones dejando nombre y WhatsApp (sin registro ni verificación) y deja reseñas si canjeó un cupón |
| **Dueño/Admin de local** | Contrata un plan, se registra, completa su perfil, administra sedes y servicios, define beneficios y cupo mensual, canjea cupones, ve métricas |
| **Super Admin** | Confirma pagos, aprueba/pausa/elimina locales, marca perfiles verificados, modera reseñas, resuelve incidencias |

### 1.4 Objetivos del MVP
1. Un usuario encuentra un local cercano y obtiene un cupón en menos de 3 clics.
2. Un local canjea ese cupón en segundos desde el celular.
3. Un local solo es visible si tiene suscripción vigente (o es piloto).
4. Desplegable con `docker compose up` y con CI verde.

### 1.5 Fuera de alcance del MVP (post-MVP)
- Reserva con estados y agenda con cupos o capacidad: la reserva se resuelve por WhatsApp.
- Integración por API de Mercado Pago (webhooks, reconciliación): el cobro es automático por el plan de suscripción, pero el registro en MOVO es manual.
- Subida y revisión de DNI: en el MVP la verificación es manual.
- Benchmarking de zona y alertas de "alta demanda" y "franjas con poca demanda".
- Notificaciones push, email o WhatsApp automáticos; reseteo de contraseña por email.
- Respuesta del dueño a las reseñas.
- Seguimiento de ubicación en vivo (se usa pin fijo).
- App móvil nativa, pagos del usuario final, réplicas de lectura, multi-región, Kubernetes.
- Favoritos ("Guardados") del cliente.

### 1.6 Métricas de éxito (piloto)
- 5–10 locales de Almafuerte/Río Tercero activos.
- ≥ 50 cupones generados y ≥ 30 % canjeados en el primer mes.
- p95 de búsqueda < 300 ms.
- 0 cupones canjeados dos veces.

---

## 2. Stack tecnológico

| Capa | Elección | Justificación / nota |
|---|---|---|
| Backend | Go 1.22+ (`net/http` + `chi`) | Idiomático, poco acoplamiento al framework |
| Acceso a datos | `pgx` + `sqlc` | SQL explícito, tipado, sin magia de ORM |
| Migraciones | `goose` (SQL puro) | Versionadas en el repo, up/down en un archivo |
| DB | PostgreSQL 16 + **PostGIS** | Búsqueda por cercanía con índice espacial |
| Tareas diarias | Loop simple en `cmd/worker` con lock consultivo de Postgres | Sin cola de jobs en el MVP; visibilidad y vencimientos se calculan al leer |
| Caché | Redis (**Fase 7**, cuando haya métricas) | Cache-aside con TTL; también rate limiting distribuido |
| Frontend | Next.js (App Router) + Tailwind | SSR para SEO de fichas de locales |
| Mapas | React Leaflet + tiles configurables por variable de entorno | Piloto: tiles públicos de OSM; producción: Protomaps (PMTiles) o proveedor (sección 12.3) |
| Storage | S3 compatible: RustFS en local, gestionado (R2/B2/S3) en prod | URLs prefirmadas |
| Cobros | Planes de suscripción de Mercado Pago (links por env) | Sin API ni webhooks en el MVP |
| Contacto | Enlace `wa.me` con mensaje prearmado | Sin Twilio ni WhatsApp Business |
| Infra | Docker + docker compose; VPS al inicio | Simple y barato |
| CI | GitHub Actions | lint, tests, build, chequeo de migraciones |

> **Storage:** el código no habla con un servidor de objetos concreto, habla con el puerto `Storage` sobre la **API S3**. El servidor es un detalle de entorno: en desarrollo corre un S3 local (RustFS, con la imagen pineada), en producción un servicio gestionado (Cloudflare R2, Backblaze B2 o AWS S3). Cambiar de servidor es cambiar configuración, no código.

> Si preferís Gin (como en otros proyectos) es válido; lo importante es que los handlers HTTP sean una capa fina y reemplazable.

---

## 3. Arquitectura

### 3.1 Estilo: monolito modular (no microservicios)
Un solo repositorio, un solo despliegue, con **módulos con fronteras estrictas**. Se puede extraer un módulo a servicio más adelante si hay una razón real.

Binarios, misma base de código:
- `cmd/api`: servidor HTTP stateless (JWT), escalable detrás de un balanceador.
- `cmd/worker`: tareas diarias mínimas (marcado de cupones vencidos, alerta de cupones sin canjear). Se crea cuando la primera de esas tareas lo requiera.
- `cmd/migrate`: herramienta de un solo uso que aplica las migraciones SQL embebidas (`goose`) con `up`, `down`, `status` y `version`. Se ejecuta antes de desplegar, no queda corriendo.

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
│   │   ├── platform/          # config, logger, db, httpx, clock, id
│   │   ├── identity/          # users, auth, roles
│   │   ├── business/          # negocios, sedes, servicios, categorías, horarios, imágenes, planes y entitlements
│   │   ├── search/            # búsqueda geoespacial y destacados del Home (solo lectura)
│   │   ├── benefits/          # beneficios y cupones
│   │   ├── reviews/           # reseñas verificadas por canje
│   │   ├── billing/           # suscripciones manuales (vigencia y gracia)
│   │   ├── metrics/           # contadores agregados y alertas
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
5. Los servicios externos reales (geocoding, storage) siempre van detrás de una interfaz con un fake para tests. Mercado Pago **no** se integra por API en el MVP.

### 3.4 CQRS ligero (sin sobreingeniería)
- Comandos y queries separados en `app/commands` y `app/queries`.
- Las queries pueden usar un repositorio de lectura propio (SQL con joins directos o una vista SQL).
- **La caché es un decorador** de la interfaz de query (`CachedBusinessSearch` envuelve a `BusinessSearch`). Se agrega en Fase 7 sin tocar el dominio.
- No hay event sourcing ni bus de mensajes externo; los eventos de dominio son llamadas en proceso.

### 3.5 Concurrencia y tareas
- `context.Context` en toda la cadena, con timeouts y cancelación en el apagado (`graceful shutdown`).
- Las tareas diarias corren en `cmd/worker` con un lock consultivo de Postgres para que no se ejecuten dos veces si hay más de una instancia.
- Nada depende de goroutines sueltas para trabajo que no se pueda perder.

### 3.6 Stateless
- Sin sesiones en memoria. Auth con JWT de acceso corto (≈15 min) + refresh token rotativo guardado (hasheado) en DB, **solo para dueños/admins de local y super admin**. Los clientes no tienen cuenta: acceden a sus cupones con un enlace con token no adivinable (ver 5.2).
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
| Webhooks de Mercado Pago | ❌ | Post-MVP, cuando la revisión mensual deje de ser razonable (≈15–20 locales) |

> Un local en Almafuerte/Río Tercero no necesita réplicas ni CDN. Diseñar *preparado* para escalar (stateless, interfaces) alcanza; implementarlo antes es costo sin retorno.

---

## 4. Modelo de datos (alto nivel)

Todas las tablas con `id` (UUID v7 o `bigserial`), `created_at`, `updated_at`. Migraciones en SQL puro.

| Tabla | Campos clave |
|---|---|
| `users` | email (único), password_hash (argon2id), role (`business_admin`/`super_admin`) — solo personal de locales y admins |
| `refresh_tokens` | user_id, token_hash, expires_at, revoked_at, replaced_by |
| `customers` | name, phone (WhatsApp, normalizado), created_at — contacto declarado, **sin verificación ni contraseña** |
| `categories` | slug, name — seed: gimnasios, pilates, yoga, entrenadores, estética, natación, pádel; y `trekking-outdoor` ("Trekking y outdoor"), agregada en una migración aparte. En esa categoría la **sede es el punto de encuentro** de la salida y **cada salida se carga como un servicio** (con sus días y horarios) |
| `businesses` | owner_user_id (único: un dueño, un negocio), category_id, name, slug (único), description (≤ 500 caracteres), whatsapp_number (solo dígitos con código de país, validado), status (`draft`/`pending_review`/`active`/`paused`/`deleted`), pilot, verified_at, verified_by, mp_payer_email (email con el que pagó en Mercado Pago, obligatorio al enviar a revisión), mp_payer_name (nombre completo del titular, obligatorio, mínimo 3 caracteres), mp_return_params (parámetros de la URL de retorno, solo pista para el admin) |
| `branches` | business_id, name, address, city, reference (opcional, ≤ 100), `location geography(Point,4326)`, external_booking_url (opcional), rating_avg, rating_count, active — **la búsqueda se hace por sede** |
| `business_hours` | branch_id, weekday, opens_at, closes_at |
| `business_images` | business_id, object_key, position |
| `services` | business_id, branch_id (opcional), name, description, price (opcional, pesos enteros; sin duración porque el MVP no tiene agenda), active — lo que el negocio ofrece (clase de yoga, alquiler de cancha, etc.) |
| `service_schedules` | service_id, weekday, starts_at, ends_at |
| `subscriptions` | business_id, plan (`starter`/`pro`, lo fija el super admin al confirmar el pago, nunca el dueño), paid_until (vigencia inicial que elige el admin al activar), mp_subscription_ref (opcional: referencia de la suscripción en el panel de Mercado Pago, la que figura en el mail "nuevo suscriptor"), confirmed_by, confirmed_at, notes |
| `benefits` | business_id, type (`percent`/`welcome_gift`), percent (10/15/20), gift_description, active, monthly_limit (configurable por el dueño) |
| `benefit_codes` | code (único), benefit_id, business_id, customer_id, access_token_hash, status (`pending`/`redeemed`/`expired`), expires_at (= creación + 48 h), redeemed_at, redeemed_branch_id, redeemed_by |
| `reviews` | branch_id, business_id, benefit_code_id (único), customer_id, rating 1–5, comment (≤ 500, opcional), display_name, show_name, status (`visible`/`reported`/`hidden`), created_at |
| `metric_daily` | business_id, branch_id (nullable), service_id (nullable), date, metric (`profile_view`/`service_view`/`whatsapp_click`), count — contadores agregados, **sin datos personales**. La forma la fija un CHECK: `profile_view` va con `branch_id` y `service_id` nulos (la ficha es por negocio); `whatsapp_click` exige `branch_id` (el servicio es opcional); `service_view` exige ambos. El día se cuenta en hora de Argentina (UTC-3) |
| `alerts` | business_id, type, payload (jsonb), created_at, read_at |
| `audit_log` | actor_id, action, entity, entity_id, metadata, created_at |

Los planes **no son una tabla**: se definen por configuración (precio para mostrar, URL del plan de Mercado Pago y límites), ver 4.2 y 7.5.

**Baja lógica.** Las sedes y los negocios no se borran: se dan de baja de forma lógica (`branches.active = false`, `businesses.status = 'deleted'`), porque las métricas, los canjes y las reseñas los referencian y son historial que el local y el admin siguen leyendo. En la base, `metric_daily` apunta a `branches` con `ON DELETE RESTRICT`, así que un `DELETE` físico de una sede con historial falla en lugar de borrar sus contadores en silencio. En la API, `DELETE /me/business/branches/{id}` desactiva la sede (es idempotente y conserva el historial); volver a mostrarla es `PUT …/active`, que revisa el límite del plan.

Índices importantes:
- `GIST (location)` en `branches`.
- `UNIQUE (code)` en `benefit_codes`; índice por `(customer_id, status)` y `(business_id, status)`.
- Índice único parcial para la regla *un cupón pendiente por cliente y local*: `UNIQUE (customer_id, business_id) WHERE status = 'pending'`.
- `UNIQUE (benefit_code_id)` en `reviews`.
- `UNIQUE NULLS NOT DISTINCT (business_id, branch_id, service_id, date, metric)` en `metric_daily` (upsert con incremento). `NULLS NOT DISTINCT` es necesario: con un `UNIQUE` común, los `NULL` de `branch_id` y `service_id` se considerarían distintos y el upsert crearía una fila nueva por cada vista.

### 4.1 Regla de visibilidad
La visibilidad **se calcula al leer**, no se guarda. Una vista SQL (`visible_branches`) expone las sedes visibles: un local aparece en la búsqueda pública si y solo si:
```
business.status = 'active'
AND (business.pilot = true
     OR today <= subscription.paid_until + SUBSCRIPTION_GRACE_DAYS)
```
Con `SUBSCRIPTION_GRACE_DAYS = 7` por defecto.

> **Evolución de la vista.** En la Fase 3 (tarea 3.1) `visible_branches` nace en una versión mínima: sede con `active = true` de un negocio con `status = 'active'`. Durante el piloto (Fases 0 a 4) todo negocio aprobado es visible, y `pilot` y el pago no intervienen. La Fase 5 (tarea 5.4) la extiende con una migración `CREATE OR REPLACE VIEW` que agrega la condición de arriba, sin cambiar las columnas, así que las consultas no cambian.

### 4.2 Planes y entitlements

Cualquier negocio (comercio o profesional) puede contratar cualquier plan; el tipo de negocio no restringe la elección.

| Característica | Starter ($19.800/mes) | Pro ($29.800/mes) |
|---|---|---|
| Perfil verificado | ✅ | ✅ |
| Servicios activos | hasta 3 | ilimitados |
| Horarios por servicio | según servicios | ilimitados |
| Sedes | 2 | 4 |
| Botón de WhatsApp (reservar clases) | ✅ | ✅ |
| Ubicación en el mapa (pin fijo) | ✅ | ✅ |
| Cupones de descuento (10/15/20 % o regalo de bienvenida) | ✅ | ✅ |
| Reseñas y puntuación | ✅ | ✅ |
| Métricas | esenciales | **inteligencia comercial** |
| Soporte | estándar | prioritario |
| Visibilidad destacada (prioridad en el inicio, búsquedas y pin destacado en el mapa) | — | ✅ |

Pro incluye todo lo de Starter.

Reglas de implementación:
- **Límites desde configuración.** Los límites (servicios, sedes) se leen de variables de entorno a través de un puerto `PlanCatalog` (hoy `ConfigPlanCatalog`; mañana podría ser un `DBPlanCatalog` editable desde el panel sin tocar el dominio). El dominio consume un value object `Entitlements` (ej. `ent.CanAddBranch(current)`); **nunca** se pregunta `if plan == "pro"`.
- Los límites se validan al **crear o activar**, no al listar.
- **Downgrade o baja de límites** (Pro → Starter, o cambio de variable): no se borra nada. Se mantienen activos los elementos más antiguos hasta el nuevo límite y el resto pasa a `inactive`, con aviso al dueño.
- **Visibilidad destacada (Pro):**
  - Sello "Destacado" claramente etiquetado en tarjetas y ficha.
  - **Inicio:** sección "Destacados cerca tuyo" solo con locales Pro dentro del radio, con rotación entre ellos para repartir la exposición.
  - **Búsquedas:** el orden es por tramo de distancia (ej. cada 2 km), luego destacado, luego distancia exacta; así un Pro lejano no le gana a un local muy cercano.
  - **Mapa:** marcador distintivo para el pin destacado.
- **Perfil verificado (MVP):** lo marca el super admin a mano (`verified_at`, `verified_by`) después de comprobar la identidad por su cuenta (en persona o videollamada). La subida del DNI es post-MVP. El sello se puede revocar.
- **Botón de WhatsApp:** el dueño carga su número al registrarse. El botón pasa por `GET /r/wa/{branch_id}?service=`, que incrementa el contador `whatsapp_click` y redirige a `https://wa.me/<número>?text=<mensaje prearmado>` (ej. "Hola, vi tu perfil en MOVO y quiero reservar Yoga"). El número no se expone en la API pública. En la API la ruta es `/api/v1/r/wa/{branch_id}` y responde `302` con `Cache-Control: no-store`. Solo redirige sedes visibles (si no, `404`); un `service` que no es del local, está inactivo o es de otra sede no es un error: se usa el mensaje genérico ("…quiero hacer una consulta") y el clic se cuenta sin servicio. Los bots se redirigen pero no se cuentan, y un fallo del contador nunca impide la redirección. `HEAD` devuelve el mismo `302` sin contar (los previsualizadores de enlaces lo usan) y la respuesta lleva `Referrer-Policy: no-referrer`.
- **Soporte prioritario:** es principalmente un proceso (canal y tiempos de respuesta definidos); en el sistema se refleja con una marca de prioridad en la bandeja del admin.

### 4.3 Métricas
Principio: **contar, no seguir**. No se guardan IP, user-agent ni identificadores; se ignoran los bots.

- **Starter, "Métricas esenciales":**
  1. Vistas al perfil (`profile_view`).
  2. Cupones generados (desde `benefit_codes`).
  3. Cupones canjeados (desde `benefit_codes`).
  4. Actividades más demandadas (vistas del servicio + clics a WhatsApp por servicio).
- **Pro, "Inteligencia comercial" (incluye todo Starter):**
  5. **Tasas de conversión** (vistas → cupones, cupones → canjes) con tendencia contra el período anterior y desglose por sede y actividad. No hay "embudo" como función aparte.
  6. **Una alerta: "Cupones sin canjear"** (la tasa de canje de los últimos 30 días queda bajo un umbral, con una cantidad mínima de cupones generados). Se muestra en el panel, como máximo una por semana.
- **Las vistas de perfil son por negocio.** La ficha `/locales/[slug]` es del negocio, así que `profile_view` se guarda sin sede. El desglose por sede de Pro aplica a los clics a WhatsApp, las vistas de servicio y los canjes (`redeemed_branch_id`). La tasa "vistas → cupones" se calcula solo a nivel negocio; "cupones → canjes" sí admite desglose por sede.
- **Post-MVP:** benchmarking anónimo de zona (solo con ≥ 5 locales activos por categoría y zona; solo medianas y percentiles) y las alertas "Alta demanda" y "Franjas con poca demanda". La demanda por actividad se mide con los contadores de servicio; MOVO no conoce ocupación ni capacidad.
- Los contadores se registran **desde la Fase 3** para tener historial cuando se construya el panel.

---

## 5. Flujos principales

### 5.1 Búsqueda por cercanía y Home
1. El cliente (sin cuenta) envía `lat`, `lng`, `radius_m`, `category`, paginación (keyset o limit/offset acotado).
2. Query sobre `visible_branches` con `ST_DWithin(location, point, radius)`, ordenada por tramo de distancia → destacado → distancia exacta.
3. Respuesta con DTO liviano (sin campos internos).
4. **Home:** además de "Cerca tuyo", una query de "Destacados cerca tuyo" (solo Pro dentro del radio, con rotación).

### 5.2 Generación y canje de cupones
**Generar** (cliente sin cuenta)
1. El cliente elige un beneficio de un local visible y completa **nombre y WhatsApp** (obligatorios). Sin verificación.
2. Se valida: beneficio activo, **un solo cupón pendiente por cliente y local** (índice único parcial), y que el local no haya superado su `monthly_limit` (tope de cupones generados por local por mes).
3. Código aleatorio criptográficamente seguro (`crypto/rand`), alfabeto sin caracteres ambiguos (sin `0/O/1/I`), 8 caracteres; si hay colisión por `UNIQUE`, se reintenta.
4. Se guarda `pending` con `expires_at = now() + 48 h` y se devuelve el código en pantalla junto con un enlace de acceso (token no adivinable, se guarda solo su hash) para volver a verlo.
5. **Anti-abuso (porque no hay verificación):** límite de intentos por IP (en memoria, interfaz lista para Redis), normalización del teléfono para detectar duplicados y el cupo mensual del local como tope de daño. Si aparece abuso real se agrega captcha o verificación.

**Canjear** (el `business_id` sale del token/permisos del admin, **nunca** del body)
```sql
UPDATE benefit_codes
SET status = 'redeemed', redeemed_at = now(), redeemed_by = $1, redeemed_branch_id = $4
WHERE code = $2
  AND business_id = $3
  AND status = 'pending'
  AND expires_at > now()
RETURNING id;
```
Si no devuelve filas → inválido, vencido, ya canjeado o de otro local. Un solo `UPDATE` atómico evita doble canje por concurrencia.

El vencimiento se evalúa por `expires_at` al leer; una tarea diaria marca como `expired` en bloque los vencidos.

### 5.3 Contacto y reserva por WhatsApp
1. En la ficha del local y en cada servicio hay un botón "Reservar por WhatsApp".
2. El botón pasa por `/r/wa/{branch_id}?service=`, que cuenta el clic y redirige a `wa.me` con un mensaje prearmado.
3. La conversación y la confirmación ocurren por fuera de MOVO, en el WhatsApp del dueño. Esa es también su "notificación" de nueva consulta.
4. Clubes con sistema propio de reservas (ej. pádel) pueden cargar `external_booking_url` para mostrar un botón adicional "Reservar cancha".

### 5.4 Alta y suscripción con Mercado Pago (cobro automático, registro manual)
1. La landing muestra los planes con botones "Contratar Starter" y "Contratar Pro". Cada botón abre el **link del plan de suscripción de Mercado Pago** (`MP_STARTER_URL`, `MP_PRO_URL`).
2. El dueño abre el link una vez, elige el medio de pago y autoriza: **Mercado Pago cobra automáticamente cada mes**.
3. Tras pagar, Mercado Pago redirige a `/onboarding` (a verificar en la configuración del plan; si no se pudiera, el admin envía el enlace de `/onboarding` por WhatsApp al confirmar el pago).
4. En `/onboarding` el dueño crea su cuenta, carga su número de WhatsApp y completa sus datos (negocio, sede con pin en el mapa, servicios, horarios, fotos). Mercado Pago no entrega comprobante ni número de operación al pagador, así que en el último paso declara, obligatorios, el **email con el que pagó** (`mp_payer_email`) y el **nombre completo del titular** (`mp_payer_name`). El onboarding **no pregunta qué plan pagó**. Si la URL de retorno trae un parámetro `plan`, y cualquier otro que Mercado Pago agregue, se guarda solo como pista visible para el admin (`mp_return_params`); **nunca es un dato confiable ni prueba de pago**. El super admin cruza el email y el nombre declarados con el mail "nuevo suscriptor" de Mercado Pago y **fija el plan al activar**.
5. El negocio queda en `pending_review`.
6. Al cobrador, Mercado Pago le envía el mail "¡Tienes un nuevo suscriptor en <plan>!" con nombre completo, email, plan, monto y frecuencia del suscriptor y un enlace a "Detalles de la suscripción": **ese mail funciona como alerta** de que hay un alta para revisar. Con los tres chequeos hechos (mail con el mismo email, plan y monto correctos, perfil revisado), el super admin activa el negocio: elige el plan y la vigencia inicial y se crea la `subscription` con `paid_until`.
7. **Cada mes** el super admin revisa en Mercado Pago qué locales cobraron y presiona "Registrar pago", que extiende `paid_until`. Si un cobro falla o el dueño cancela, el local sigue visible durante la gracia (7 días) y luego deja de aparecer.
8. El panel muestra los locales "por vencer" (7 días) y arma un enlace `wa.me` con un recordatorio prearmado para el dueño.
9. Cambio de plan, cancelación y actualización de precios por inflación son acciones manuales del super admin (el precio se actualiza en el plan de Mercado Pago y en la configuración).

### 5.5 Reseñas (Fase 4C)
1. Solo puede reseñar quien **canjeó un cupón** en esa sede; **una reseña por cupón canjeado**. El cliente accede con su enlace/token.
2. El formulario solo pide puntuación (1–5) y comentario opcional (≤ 500). **No pide nombre:** se toma del cliente asociado al cupón (primer nombre + inicial del apellido, guardado en `display_name`). Con "Mostrar mi nombre" desactivado se publica como "Cliente de MOVO".
3. Se muestra como "Reseña verificada por canje". Plazo: 30 días desde el canje.
4. Se muestran promedio y cantidad (`rating_avg`, `rating_count` en `branches`, actualizados por evento de dominio). Con menos de 3 reseñas se muestra "Nuevo".
5. El local puede **reportar** una reseña; solo el super admin puede **ocultarla**. El dueño no borra reseñas.
6. La puntuación sirve como filtro opcional (puntuación mínima) y **no** se usa para ordenar.
7. La API pública nunca expone teléfono; solo el super admin ve el vínculo con el cliente, para moderar. Ante un pedido de borrado de datos se anonimiza `display_name`.

### 5.6 Actividad en el panel (notificaciones dentro de la app)
- No hay push, email ni WhatsApp automáticos.
- El panel del dueño tiene **"Actividad reciente"** (cupones generados, canjes, reseñas nuevas) con contador de novedades, consultado cada 30–60 segundos mientras el panel está abierto (`GET /me/business/activity`).
- El canje lo realiza el propio local, por eso no hay alerta de canje.
- El reseteo de contraseña es asistido: el super admin genera un enlace temporal y se lo envía al dueño por WhatsApp.

### 5.7 Panel Super Admin
- Alta, pausa, reactivación y baja lógica de locales; marcar verificado; marcar piloto (siempre con `audit_log`).
- Suscripciones: confirmar pago, registrar pago mensual, suspender, cambiar de plan, ver "por vencer".
- Moderación de reseñas y reseteo de contraseña asistido.

---

## 6. API (contrato inicial, `/api/v1`)

| Método | Ruta | Rol | Descripción |
|---|---|---|---|
| POST | `/auth/register` · `/auth/login` · `/auth/refresh` · `/auth/logout` | público | Autenticación de dueños |
| GET | `/plans` | público | Planes con precio y URL de contratación |
| GET | `/search/businesses` | público | Búsqueda geoespacial |
| GET | `/home/featured` | público | Destacados cerca tuyo |
| GET | `/businesses/{slug}` | público | Ficha del local |
| GET | `/r/wa/{branch_id}` | público | Cuenta el clic y redirige a WhatsApp |
| POST/PUT | `/me/business` | business_admin | Crear/editar mi negocio |
| POST/PUT/DELETE | `/me/business/branches` | business_admin | Sedes (según límite del plan) |
| POST/PUT | `/me/business/services` | business_admin | Servicios y horarios (según límite del plan) |
| POST | `/me/business/images/presign` | business_admin | URL prefirmada de subida |
| POST/PUT | `/me/business/benefits` | business_admin | Beneficios y cupo mensual |
| GET | `/me/business/activity` | business_admin | Actividad reciente |
| GET | `/me/business/metrics` | business_admin | Métricas según plan |
| POST | `/benefits/{id}/codes` | público (rate limit) | Generar cupón con nombre y WhatsApp |
| GET | `/access/{token}` | token del cliente | Ver mis cupones |
| POST | `/access/{token}/reviews` | token del cliente | Dejar reseña |
| GET | `/branches/{id}/reviews` | público | Reseñas paginadas |
| POST | `/me/business/reviews/{id}/report` | business_admin | Reportar reseña |
| POST | `/me/business/codes/redeem` | business_admin | Canjear cupón |
| * | `/admin/...` | super_admin | Locales, verificación, suscripciones, moderación, reseteo de contraseña |

Convenciones: errores en el sobre JSON único de §7.2 (`code`, `message`, `details`), validación de entrada en la capa HTTP, paginación consistente, OpenAPI generado o mantenido en `docs/openapi.yaml`.

---

## 7. Requisitos no funcionales

### 7.1 Seguridad
- Contraseñas con argon2id (o bcrypt con costo adecuado); rate limiting en login, generación y canje.
- JWT firmado con rotación de claves planificada; refresh tokens hasheados y rotativos.
- Autorización por recurso (un admin solo toca **su** local): se verifica `owner_user_id`/`business_id` en cada comando.
- Validación de uploads (tipo MIME real, tamaño máx.), URLs prefirmadas con expiración corta.
- Clientes sin cuenta: tokens de acceso aleatorios de ≥ 128 bits guardados hasheados, con expiración; rate limit en endpoints públicos que escriben.
- El número de WhatsApp del dueño no se expone en la API pública (solo en el redirect).
- Confirmación de pagos solo por el super admin; nada que llegue por URL de retorno se toma como prueba de pago ni define el plan.
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

### 7.5 Configuración (variables de entorno)
```
# Planes y límites
PLAN_STARTER_MAX_BRANCHES=2
PLAN_STARTER_MAX_ACTIVE_SERVICES=3
PLAN_PRO_MAX_BRANCHES=4
PLAN_PRO_MAX_ACTIVE_SERVICES=0        # 0 = ilimitado
PLAN_STARTER_PRICE=19800
PLAN_PRO_PRICE=29800
MP_STARTER_URL=                       # link del plan de suscripción de Mercado Pago
MP_PRO_URL=
SUBSCRIPTION_GRACE_DAYS=7

# Búsqueda y cupones
SEARCH_DISTANCE_BUCKET_M=2000         # tramo de distancia para ordenar destacados
CODE_TTL_HOURS=48

# Reseñas y alertas
REVIEW_WINDOW_DAYS=30
REVIEW_MIN_COUNT_FOR_RATING=3
ALERT_MIN_CODES=20
ALERT_MIN_REDEEM_RATE=0.2
```
Todas las instancias de la API deben tener los mismos valores; cambiarlos requiere reiniciar. Se validan al arrancar (el proceso falla si faltan o son inválidos).

---

## 8. Estrategia de testing

| Nivel | Qué cubre | Herramientas |
|---|---|---|
| Unitarios | Reglas de dominio (generación de cupón, visibilidad, vigencia y gracia, entitlements, reglas de reseñas) | `testing`, `testify` |
| Integración | Repositorios y queries reales (PostGIS incluido), adaptador de Storage contra el S3 local | `testcontainers-go` |
| Fakes | Storage y Geocoder con implementaciones fake | interfaces + fakes |
| Concurrencia | Canje simultáneo del mismo cupón (debe canjear exactamente 1) | tests con `errgroup` |
| API | Endpoints con DB de test | `httptest` |
| Frontend | Hooks y componentes clave | Vitest + Testing Library |
| E2E (mínimo) | Buscar → generar cupón → canjear | Playwright |

Meta de cobertura: ≥ 70 % global y ≥ 90 % en `domain` y `billing`.

---

## 9. Plan por fases

Convenciones de trabajo:
- **Una fase por prompt.** Las fases 4 y 5 pueden partirse en dos mitades.
- **Una tarea = un commit pequeño y revisable** (idealmente < 400 líneas), en la rama de la fase, con la tarea tildada en este documento.
- **El estado es este documento:** la primera tarea sin tildar es por donde se sigue.
- **Vistas y backend juntos:** cada fase lista las vistas que necesita; no se inventa ninguna vista, se pide la imagen del diseño al llegar a su tarea.
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
- [x] 0.8 Frontend Next.js + Tailwind base, ESLint/Prettier, layout (S)

**DoD:** `make up` levanta todo; CI verde; `/readyz` responde con la DB conectada.

---

### Fase 1 — Identidad y autorización (dueños y super admin)
**Objetivo:** login seguro para dueños de local y super admin. Los clientes no tienen cuenta.

- [x] 1.1 Migraciones `users`, `refresh_tokens` (S)
- [x] 1.2 Dominio `identity`: entidad User, validaciones, hashing (M)
- [x] 1.3 Casos de uso Register / Login / Refresh / Logout (M)
- [x] 1.4 Middleware JWT + middleware de roles (S)
- [x] 1.5 Límite de intentos de login (en memoria, interfaz lista para Redis) (S)
- [x] 1.6 Primer super admin por seed/variables de entorno (S)
- [x] 1.7 Frontend: login y registro de dueños, manejo de sesión en hooks (M)

**Vistas:** Login y registro del dueño.
**DoD:** tests unitarios + integración de auth; un dueño no accede a rutas de super admin ni a datos de otro negocio.

---

### Fase 2 — Negocios, servicios y onboarding (sin cobro todavía)
**Objetivo:** que un local pueda existir, completar su perfil y ser aprobado manualmente.

- [x] 2.1 Migraciones `categories` (con seed: gimnasios, pilates, yoga, entrenadores, estética, natación, pádel; luego se sumó `trekking-outdoor` con la migración 00011, sin tocar el seed original), `businesses`, `branches`, `business_hours`, `business_images` (con PostGIS) (S)
- [x] 2.2 Dominio `business`: estados, slug, validación y normalización del número de WhatsApp (M)
- [x] 2.3 Casos de uso: crear/editar mi negocio, agregar/editar sedes y horarios (un dueño = un negocio) (M)
- [x] 2.4 Puerto `Storage` + adaptador S3 (RustFS en local, R2/B2/S3 en prod) con URL prefirmada y validación; endpoint, región y modo path-style configurables; test de integración (tag `integration`) que sube y baja un objeto con URL prefirmada contra el servidor local (M)
- [x] 2.5 Puerto `PlanCatalog` + `ConfigPlanCatalog` (env), value object `Entitlements` y política de downgrade sin borrar datos (M)
- [x] 2.6 Migraciones y dominio de `services` y `service_schedules` (CRUD, activar/desactivar, límites del plan) (M)
- [x] 2.7 Super admin: aprobar / pausar / reactivar / baja lógica, marcar verificado, marcar piloto + `audit_log` (M)
- [x] 2.7b Capa HTTP de la fase: endpoints `/me/business...`, `/admin/businesses...` y `/plans`, con rol por grupo de rutas, y cableado en `cmd/api` (M)
- [x] 2.8 Frontend `/onboarding`: crear cuenta y completar datos, WhatsApp, sede con pin en el mapa, fotos y, en el paso 4, email y nombre completo del titular en Mercado Pago (obligatorios; sin número de operación ni comprobante); estado "en revisión" (L)
- [x] 2.9 Frontend: Servicios y horarios y Sedes, con contador del límite del plan (M)
- [x] 2.10 Frontend super admin: lista de locales y acciones; en el cajón, email y nombre declarados, pista de plan si existe y checklist de confirmación ("Mail de Mercado Pago con el mismo email", "Plan y monto correctos", "Perfil revisado") (M)

**Vistas:** Onboarding, Servicios y horarios, Sedes, Super admin → Locales.
**DoD:** un dueño crea su local, el admin lo activa y lo marca verificado, y queda en estado `active`.

---

### Fase 3 — Búsqueda, mapa y contacto por WhatsApp
**Objetivo:** el usuario encuentra locales cercanos y contacta al local.

- [x] 3.1 Crea la versión mínima de `visible_branches` (sede activa de un negocio `active`; sin pilot ni pago) y la query sobre ella con `ST_DWithin`, orden por tramo de distancia → destacado → distancia exacta, y filtros por categoría (M)
- [x] 3.2 Endpoint de búsqueda con paginación y DTO liviano (S)
- [x] 3.3 Query y endpoint `/home/featured` con rotación entre locales Pro (S)
- [x] 3.4 Seed de 10k locales falsos y medición p95 con `EXPLAIN ANALYZE` (S)
- [x] 3.5 Puerto `Geocoder` con fake, adaptador Nominatim con limitador (1 req/s) + caché y User-Agent identificatorio; tiles configurables por env (M)
- [x] 3.6 Contadores `metric_daily` (migración, upsert, filtrado básico de bots, sin datos personales) (M)
- [x] 3.7 Endpoint `GET /r/wa/{branch_id}?service=` que cuenta el clic y redirige a `wa.me` con mensaje prearmado (S)
- [ ] 3.8 Frontend: Home y Mapa (React Leaflet, marcadores, geolocalización del navegador) (L)
- [ ] 3.9 Frontend: ficha SSR del local `/locales/[slug]` con metadatos SEO, servicios, horarios, sello de verificado y botón de WhatsApp (M)

**Vistas:** Home, Mapa, Ficha del local (opcionales: Explorar, Splash).
**DoD:** búsqueda p95 < 300 ms; la ficha se renderiza del lado del servidor; el botón de WhatsApp abre la conversación y suma el contador.

---

### Fase 4 — Beneficios y cupones (el lazo de valor)
**Objetivo:** generar y canjear cupones de forma segura y atómica, sin cuenta de cliente.

- [ ] 4.1 Migraciones `customers`, `benefits`, `benefit_codes` + índice único parcial (S)
- [ ] 4.2 Dominio `benefits`: tipos (`percent` 10/15/20 o `welcome_gift`), TTL de 48 h, cupo mensual, estados (M)
- [ ] 4.3 CRUD de beneficios y cupo mensual desde el panel del local (M)
- [ ] 4.4 Comando `GenerateCode`: nombre y WhatsApp, un pendiente por cliente y local, cupo, reintento por colisión, token de acceso (M)
- [ ] 4.5 Anti-abuso: límite por IP en memoria + normalización de contacto (S)
- [ ] 4.6 Comando `RedeemCode` atómico + test de concurrencia (M)
- [ ] 4.7 Query `GET /access/{token}` (S)
- [ ] 4.8 Vencimiento de cupones al leer y marcado diario en bloque (S)
- [ ] 4.9 `GET /me/business/activity` (cupones generados, canjes) (S)
- [ ] 4.10 Frontend del dueño: Beneficios y cupo, Panel de inicio con actividad, Canjear cupón (L)
- [ ] 4.11 Frontend del cliente: Obtener cupón, Cupón generado, Mis cupones (L)

**Vistas:** Beneficios y cupo mensual, Panel de inicio, Canjear cupón, Obtener cupón, Cupón generado, Mis cupones.
**DoD:** test de concurrencia demuestra un único canje; el flujo completo funciona de punta a punta y los intentos de abuso quedan frenados por el límite por IP y el cupo.

> **Hito de piloto:** con las Fases 0–4 podés cargar 5–10 locales reales marcados como `pilot` y **validar el producto antes de activar cobros**.

---

### Fase 4C — Reseñas verificadas por canje (cortable si el tiempo aprieta)
**Objetivo:** confianza para el cliente sin abrir la puerta a reseñas falsas.

- [ ] 4C.1 Migración `reviews` + `rating_avg` y `rating_count` en `branches` (S)
- [ ] 4C.2 Dominio `reviews`: reglas (cupón canjeado, una por cupón, plazo de 30 días, nombre a mostrar) (M)
- [ ] 4C.3 Comando crear reseña por token de acceso (M)
- [ ] 4C.4 Query de listado paginado (S)
- [ ] 4C.5 Actualización de agregados por evento de dominio (S)
- [ ] 4C.6 Reportar (dueño) y ocultar (super admin) con `audit_log` (M)
- [ ] 4C.7 Filtro de puntuación mínima en la búsqueda (S)
- [ ] 4C.8 Frontend: estrellas en tarjetas y ficha, Dejá tu reseña, Reseñas del local, moderación en el admin (L)

**Vistas:** Dejá tu reseña, Reseñas del local, moderación de reseñas.
**DoD:** solo se puede reseñar con un cupón canjeado; los agregados se actualizan al crear u ocultar; el teléfono nunca se expone.

---

### Fase 5 — Suscripción con Mercado Pago (cobro automático, registro manual)
**Objetivo:** cobrar a los locales y controlar su visibilidad sin integrar la API de Mercado Pago.

- [ ] 5.1 Migración `subscriptions` (S)
- [ ] 5.2 Dominio `billing`: vigencia, gracia y estados derivados (M)
- [ ] 5.3 Configuración de planes por env y endpoint público `GET /plans` con precio y URL de contratación (S)
- [ ] 5.4 Extiende `visible_branches` (creada en 3.1) con una migración nueva (`CREATE OR REPLACE VIEW`) que agrega la condición "pilot o hoy <= paid_until + gracia"; el down restaura la versión mínima. Tests: activo, en gracia, vencido y piloto (M)
- [ ] 5.5 Super admin: confirmar pago (el admin elige el plan y la vigencia inicial; el plan nunca sale del onboarding ni de la URL de retorno), registrar pago mensual, suspender, cambiar de plan + `audit_log` (M)
- [ ] 5.6 Vista "por vencer en 7 días" y enlace `wa.me` de recordatorio (S)
- [ ] 5.7 Aplicar límites del plan al cambiarlo (downgrade sin borrar datos) (M)
- [ ] 5.8 Frontend: landing con planes y botones de Mercado Pago, Plan y suscripción del dueño (vigencia y aviso de gracia), Super admin → Suscripciones (L)

**Vistas:** Landing con planes, Plan y suscripción, Super admin → Suscripciones.
**DoD:** con los casos de vigencia (al día, en gracia, vencido, piloto, cambio de plan) el estado y la visibilidad son siempre correctos; el flujo pago → `/onboarding` → revisión → activación funciona de punta a punta.

---

### Fase 6 — Panel Super Admin y métricas
**Objetivo:** que el admin opere sin tocar la base y que el dueño vea resultados.

- [ ] 6.1 Super admin: listados y filtros de locales y suscripciones (M)
- [ ] 6.2 Reseteo de contraseña asistido (enlace temporal para enviar por WhatsApp) (M)
- [ ] 6.3 Métricas del super admin: locales activos, cupones generados y canjeados (S)
- [ ] 6.3b Anti-inflado de métricas: descartar el mismo clic o vista repetido desde la misma IP en una ventana corta, en memoria (interfaz lista para Redis) y sin persistir la IP; se aplica a `profile_view`, `service_view` y `whatsapp_click` antes de mostrar métricas en el panel (S)
- [ ] 6.4 Métricas esenciales del local (Starter) sobre `metric_daily` y las tablas de cupones (M)
- [ ] 6.5 Actividades más demandadas (vistas del servicio + clics a WhatsApp por servicio) (S)
- [ ] 6.6 Tasas de conversión con tendencia y desglose por sede y actividad (Pro). Las vistas de perfil son por negocio (§4.3): "vistas → cupones" se calcula solo a nivel negocio; el desglose por sede aplica a clics a WhatsApp, vistas de servicio y canjes (`redeemed_branch_id`) (M)
- [ ] 6.7 Alerta "Cupones sin canjear" (tarea diaria, umbral y muestra mínima por env, máximo una por semana) + bandeja en el panel (M)
- [ ] 6.8 Marca de prioridad en la bandeja del admin para locales con soporte prioritario (S)
- [ ] 6.9 Frontend: Métricas (Starter y Pro con candado), alertas, admin completo (L)

**Vistas:** Métricas, Super admin completo.
**DoD:** el admin puede resolver un problema de cobro o de cuenta sin tocar la base de datos; las métricas coinciden con los datos reales.

---

### Fase 7 — Rendimiento y escala (opcional, guiada por métricas)
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
- [ ] 8.5 Términos y condiciones (incluyendo reglas de moderación de reseñas), política de privacidad, definición de facturación (S)
- [ ] 8.6 Runbook: operación manual mensual de pagos (revisar Mercado Pago y registrar pagos), cómo confirmar un alta con los tres chequeos (el mail "nuevo suscriptor" de Mercado Pago tiene el mismo email que declaró el dueño; plan y monto correctos; perfil revisado), cómo restaurar un backup; y, para las actividades outdoor (categoría `trekking-outdoor`), confirmar que el guía esté habilitado y tenga seguro antes de activar el alta (S)

**DoD:** sistema en producción, monitoreado y con piloto activo.

---

## 10. Definition of Done global (por tarea)
- Compila, pasa lint y todos los tests.
- Tests nuevos para la lógica agregada.
- Sin secretos ni TODOs críticos.
- Migración reversible (up/down) cuando aplica.
- Documentación actualizada (README/ADR/OpenAPI) si cambia el comportamiento.
- La tarea queda tildada en este documento en el mismo commit.

---

## 11. Riesgos y mitigaciones

| Riesgo | Mitigación |
|---|---|
| Servidor público de **Nominatim** (1 req/s por aplicación, sin autocompletado) y **tiles de OSM** (no pensados para producción a escala) | Puerto `Geocoder` y tiles configurables por env; geocodificar solo al dar de alta una sede, con caché; en producción usar Protomaps/proveedor (sección 12.3) |
| Error humano al registrar pagos manualmente y falta de escala | Vista "por vencer", gracia de 7 días, runbook y automatización con webhooks cuando haya ≈15–20 locales |
| Alguien llega a `/onboarding` sin haber pagado | El negocio queda en `pending_review` y solo el super admin lo activa tras confirmar el pago |
| Doble canje de cupón | `UPDATE` atómico + test de concurrencia |
| Abuso de cupones al no haber cuenta ni verificación | Límite por IP, normalización de contacto, un pendiente por cliente y local, cupo mensual; si hay abuso real, agregar captcha o verificación por WhatsApp |
| Reseñas falsas, autoreseña del dueño y contenido de terceros | Solo con cupón canjeado, límite por IP, moderación del super admin, detección por contacto del dueño y reglas en los términos |
| Revelar la identidad de un competidor con pocos datos (benchmarking, post-MVP) | Mínimo de 5 locales por categoría y zona; solo medianas y percentiles |
| Precios fijos en ARS con inflación | Precios en configuración y plan de Mercado Pago; proceso manual de actualización documentado |
| Sobreingeniería temprana (réplicas, CDN, microservicios, webhooks) | Fase 7 solo con métricas; ADRs para cada decisión |
| Los locales no ven valor y no pagan | Piloto gratuito antes de activar cobros, medir canjes y reseñas |
| Facturación/impuestos de la suscripción | Cobrar recurrentemente requiere inscripción fiscal y facturas: definirlo con un contador antes del lanzamiento (Fase 8) |
| Datos personales de usuarios | Mínimo dato necesario, política de privacidad, Ley 25.326 |
| Borrar una sede o un negocio y perder su historial (métricas, canjes, reseñas) | Baja lógica (§4): las sedes se desactivan y los negocios pasan a `deleted`; las claves foráneas del historial son `RESTRICT` donde corresponde, así que un `DELETE` físico falla |
| Inflar las métricas: `/r/wa` y las vistas son públicos y cuentan en cada GET, y el filtro de bots solo descarta agentes automáticos | Tarea 6.3b: descartar el mismo clic o vista repetido desde la misma IP en una ventana corta, en memoria y sin persistir la IP. Hasta entonces, las métricas se rotulan como "vistas" y "clics", no como personas |

---

## 12. Decisiones

### 12.1 Decisiones tomadas
| # | Tema | Decisión |
|---|---|---|
| 1 | Producto | Cupones de descuento y contacto por WhatsApp; la reserva se resuelve por WhatsApp, sin estados ni agenda en el MVP |
| 2 | Cupones | Un cupón pendiente por cliente y local, válido 48 h; el dueño configura el cupo mensual (tope de cupones generados por local por mes) |
| 3 | Beneficios | 10/15/20 % o regalo de bienvenida, elegido por el dueño; un local puede tener todos los que quiera; valen en todas las sedes |
| 4 | Cuenta del cliente | No existe; deja nombre y WhatsApp (obligatorios) al generar un cupón; sin verificación |
| 5 | Planes | Starter $19.800 · Pro $29.800; cualquier negocio puede contratar cualquiera |
| 6 | Sedes y servicios | Un dueño = un negocio con sedes: Starter 2, Pro 4; Starter hasta 3 servicios activos, Pro ilimitados; límites por variable de entorno |
| 7 | Cobros | Plan de suscripción de Mercado Pago (cobro automático) con links por env; sin API ni webhooks; el super admin confirma el alta y registra cada pago mensual |
| 8 | Gracia | 7 días tras vencer la vigencia |
| 9 | Verificación del perfil | Manual por el super admin en el MVP; subida de DNI post-MVP |
| 10 | Mapas y geocoding | Ver 12.3 |
| 11 | Ubicación | Pin fijo en el mapa; seguimiento en vivo queda fuera |
| 12 | Visibilidad destacada (Pro) | Prioridad en el inicio, búsquedas y pin destacado en el mapa, con sello "Destacado" |
| 13 | WhatsApp | El dueño carga su número; botón `wa.me` con mensaje prearmado; sin proveedores |
| 14 | Notificaciones | Solo dentro de la app (actividad reciente con consulta periódica); sin push, email ni WhatsApp automáticos |
| 15 | Reseñas | En el MVP, solo con cupón canjeado, una por cupón, nombre tomado del cupón, moderación del super admin |
| 16 | Métricas | Contadores agregados sin datos personales; Starter esenciales, Pro tasas de conversión y una alerta; sin embudo |
| 17 | Sin captcha ni cola de jobs | Se agregan solo si aparece abuso o necesidad real |

### 12.2 Pendientes de confirmar
- **Sedes de Starter:** se interpretó "una sede más" como 2 sedes en total (principal + 1).
- **Redirección post-pago:** verificar al crear los planes si Mercado Pago permite configurar la URL de retorno a `/onboarding`; si no, el admin envía el enlace por WhatsApp.
- **Verificación con DNI:** se pospuso a después del piloto; confirmar si se mantiene así.
- **Facturación:** definir con un contador antes del lanzamiento.

### 12.3 Mapas y geocodificación (decisión recomendada)
| Necesidad | Piloto (Fases 2–4) | Producción |
|---|---|---|
| Tiles del mapa | Tiles públicos de OSM con URL en variable de entorno, atribución y caché | Protomaps (archivo PMTiles propio en el bucket) o proveedor como MapTiler |
| Geocodificar la dirección de una sede | Nominatim público, solo al dar de alta una sede, con User-Agent propio, caché y limitador global de 1 req/s | Mismo adaptador, o Nominatim/Photon autoalojado si el volumen crece |
| Ubicación del cliente | `navigator.geolocation` del navegador (sin geocoding) | Igual |
| Buscar "mi ciudad/barrio" | Lista local de localidades cargada en DB | Igual, o API Georef del Estado argentino si sigue activa |
| Autocompletado de direcciones | No (la política de Nominatim público lo prohíbe) | Photon autoalojado, solo si se justifica |

---

## 13. Post-MVP (backlog)
1. Automatizar cobros con webhooks de Mercado Pago (cuando haya ≈15–20 locales).
2. Subida y revisión de DNI con storage privado.
3. Benchmarking anónimo de zona y alertas de demanda.
4. Respuesta del dueño a reseñas.
5. Reseteo de contraseña por email y notificaciones por email/push.
6. Reserva con estados o agenda con cupos, si el piloto lo justifica.
7. Favoritos del cliente.
8. Captcha o verificación del cliente, si aparece abuso.
9. Sede principal: hoy ninguna sede se distingue de las demás ("principal + 1" en 12.2 es solo una forma de contar). Hay que poder marcar cuál es la sede principal de cada local (por ejemplo `branches.is_primary`, una sola por negocio) y usarla donde se necesite una sede única, como la ficha o la tarjeta del Home.

---

## 14. Glosario
- **B2B2C:** el negocio paga, el consumidor final usa.
- **Plan de suscripción (Mercado Pago):** link de cobro recurrente automático; el dueño lo autoriza una vez.
- **Cupón:** beneficio que el cliente genera y canjea en el local mediante un código alfanumérico (`benefit_codes`).
- **Gracia:** días posteriores al vencimiento durante los cuales el local sigue visible.
- **Idempotencia:** procesar el mismo evento dos veces produce el mismo resultado.
- **Cache-aside:** la app consulta la caché, y si falla, lee de la DB y completa la caché.
- **ADR:** Architecture Decision Record, documento corto que registra una decisión y su motivo.