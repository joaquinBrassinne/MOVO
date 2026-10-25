# Fase 2 — Negocios, servicios y onboarding

Rama: `phase-2/business-onboarding`. Estado general: `docs/sdd.md` §9 (todas las tareas 2.1 a 2.10 tildadas).

## Objetivo
Que un local pueda existir, completar su perfil y ser aprobado manualmente (sin cobro).

## Decisiones
- Un módulo no importa internals de otro, tampoco en tests (los tests de `business` crean usuarios con SQL directo).
- Descripción del negocio: máximo 500 caracteres (dominio, validación, contador de la UI y SDD).
- Modelo ampliado por pedido del usuario: `services.price` (opcional, pesos enteros) y `branches.reference` (opcional, ≤ 100). **Sin duración**: el MVP no tiene agenda.
- El slug se deriva del nombre y se sufija (`-2`, `-3`...) si está tomado; no cambia al editar el nombre.
- `Storage` es un puerto compartido en `internal/platform/storage`; SDK `aws-sdk-go-v2`. Solo JPEG/PNG/WebP, máximo 5 MiB, el tamaño va firmado en la URL. Máximo 6 fotos por negocio.
- Límites del plan: `AddBranch`, `SetBranchActive`, `AddService` y `SetServiceActive` los validan al crear o activar; solo cuentan los elementos activos. Los pilotos tienen límites Pro de forma TEMPORAL (`billing/adapters/pilotplan`, a borrar al terminar el piloto; luego pagan Pro). El resto queda en Starter (`fixedplan`) hasta la fase 5.
- Las acciones del super admin viven en el módulo `business` (reutilizan su dominio sin imports cruzados); `Apply` guarda el cambio y su entrada de `audit_log` en una transacción con bloqueo optimista por (estado, piloto, verificación).
- El navegador llama a la API por el proxy `app/api/v1/[...path]` con lista cerrada (`me/`, `admin/`, `plans`, `categories`); `auth/*` NO pasa: el refresh token solo lo ve `app/api/session` (cookie httpOnly). `API_URL` se lee en runtime (no son rewrites de next.config porque esas se fijan al construir).
- Subida de fotos: PUT directo al storage con la URL prefirmada, así que el bucket necesita CORS. En desarrollo lo aplica `S3_ENSURE_BUCKET=true` + `S3_CORS_ORIGINS=http://localhost:3000`; en producción se configura en el proveedor (R2/B2/S3).
- Geocodificación: botón explícito "Buscar en el mapa" (SDD §12.3 prohíbe autocompletar con Nominatim público), 1 consulta por segundo con caché. Campo "Ciudad" agregado porque la API lo exige y el diseño no lo tenía.
- Frontend sin runner de tests de componentes: la lógica pura (`lib/**/*.ts`) se prueba con `node --test` (`npm test`, en la CI). Los módulos puros usan imports relativos con extensión `.ts` (el runner no resuelve el alias `@/`).
- Rutas del portal: `(acceso)` marco oscuro, `onboarding`, `(panel)` con navegación inferior (`/socios/panel`, `/socios/servicios`, `/socios/sedes[/nueva|/id]`) y `/admin/locales` para el super admin (el ingreso redirige por rol).

## Variables que `cmd/api` exige
`DATABASE_URL`, `JWT_SECRET`, `PLAN_STARTER_MAX_BRANCHES`, `PLAN_STARTER_MAX_ACTIVE_SERVICES`, `PLAN_STARTER_PRICE`, `PLAN_PRO_MAX_BRANCHES`, `PLAN_PRO_MAX_ACTIVE_SERVICES`, `PLAN_PRO_PRICE`, `S3_BUCKET`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `GEOCODER_USER_AGENT`. En desarrollo además: `S3_ENDPOINT=http://localhost:9000`, `S3_PATH_STYLE=true`, `S3_ENSURE_BUCKET=true`, `S3_CORS_ORIGINS=http://localhost:3000`. `.env.example` no se pudo editar (permisos): agregarlas a mano.

## Hecho
- 2.1 a 2.10 y 2.7b. DoD verificada: un test de la API y un recorrido real (Postgres, RustFS, API y proxy web) donde el dueño crea su local, el admin lo aprueba y lo verifica, y queda `active`.
- Puertas: `make fmt-check lint test build`, y en el frontend lint, format:check, typecheck, test y build, todo en verde.

## Diferencias entre los diseños y el SDD (se siguió el SDD)
- Sin duración del servicio. Sin "Plan Trimestral / Pago online detectado / comprobante PDF" ni "Verificar en MP" (no hay integración de pagos). Consejo "4x más reservas" reemplazado por uno sin cifra.
- Sin "Pasar a Pro" ni tarjetas de upsell (el precio del diseño, $9.900, no es el del SDD y el cobro es de la fase 5). Sin "Simular estado de cuenta" (era una demo del diseño). Planes con los nombres del SDD (Starter/Pro, no "Plan Base").
- Sedes: sin miniatura de mapa, sin estado "Borrador/Incompleto" ni "canchas" (no existen en el modelo); "Ver ficha" llega con la fase 3. Estados: Activa/Oculta.
- Navegación inferior sin "Canjear" (fase 4). "Inicio" es provisorio (no tiene diseño).
- Super admin: sin "Descargar CSV", "Alta manual", "Rechazar o pedir corrección" (no hay esa transición en el SDD), filtro por plan, "Más filtros", Suscripciones/Reseñas/Métricas (fases 5 y 6) ni comprobante PDF.
- Pantalla "en revisión": sin barra "80 %" ni enlace a soporte (no hay canal definido); se conservó "Tiempo estimado < 2 hs". Botón final del paso 4: "Enviar a revisión". Chips de categoría sin íconos. "Navegación en un toque" omitido (no existe).

## Pendiente
- **Revisión visual de las vistas en el navegador** (no se pudo hacer desde la sesión).
- Datos de prueba que dejó el último recorrido real en la base local de desarrollo (el clasificador de permisos bloqueó la limpieza): usuarios `owner%@example.com`, el super admin `e2e-admin@example.com` (contraseña conocida: borrarlo) y sus negocios y entradas de `audit_log`.
- `.env.example` con las variables nuevas; `docs/openapi.yaml` (no existe, ni para identity); `make ci` no existe aunque `CLAUDE.md` lo menciona.
- Aplicar la política de downgrade (`Entitlements.*ToDeactivate`) cuando cambie el plan (fase 5) y un caso de uso para reactivar sedes inactivas por downgrade.
- Fases futuras: ficha pública (fase 3), "Canjear" (fase 4), upgrade a Pro (fase 5), tag de git de la fase.

## Pruebas E2E (Playwright)
- `cd frontend && npm run e2e` (requiere `make up`: Postgres y RustFS). Cada corrida crea la base `movo_e2e` desde cero, migra, compila la API (`backend/bin/api-e2e`), levanta un Nominatim falso y `next dev -p 3100`; no toca la base de desarrollo. El bucket es `movo-e2e`.
- 59 tests en 5 specs que son una secuencia sobre la misma base (van de a uno y en orden): `01-access`, `02-onboarding`, `03-admin`, `04-panel`, `05-removal`. Un spec suelto no corre solo.
- No debe haber otro `next dev` abierto en `frontend/` (comparte `.next`).
- No corren en CI todavía (necesitan Docker, Go y el navegador).

## Hallazgos de las E2E
- El mapa del alta se pintaba encima de la barra inferior: arreglado con `isolate`; el test `ONBOARD-E2E-010` falla sin ese cambio.
- El texto "Ingresá un link que empiece con https://" no coincide con la regla: web y API aceptan `http://`. Pendiente decidir si se exige https o se cambia el texto.
- Intermitencia sin causa confirmada: una vez de cuatro corridas completas, una navegación completa a `/socios/sedes/99999` justo después de ingresar terminó en la pantalla de ingreso. No se pudo reproducir en 45 intentos de sonda.
- La API no deduplica fotos por contenido (la clave es única por subida); el máximo es 6.
- `next dev` imprime un error de "instant navigation" para `/socios/panel` (segmento descartado); no rompe nada, pero conviene mirarlo.
