# Fase 2 — Negocios, servicios y onboarding

Rama: `phase-2/business-onboarding`. Estado general: `docs/sdd.md` §9 (la primera tarea sin tildar es por donde se sigue).

## Objetivo
Que un local pueda existir, completar su perfil y ser aprobado manualmente (sin cobro).

## Decisiones
- Un módulo no importa internals de otro, tampoco en tests (los tests de `business` crean usuarios con SQL directo).
- La descripción del negocio admite hasta 500 caracteres (decidido por el usuario; dominio, validación, contador de la UI y SDD).
- El slug se deriva del nombre y se sufija (`-2`, `-3`...) si está tomado; no cambia al editar el nombre.
- `Storage` es un puerto compartido en `internal/platform/storage`; el SDK es `aws-sdk-go-v2` (decidido por el usuario). Solo JPEG/PNG/WebP, máximo 5 MiB, el tamaño va firmado en la URL.
- Se suma la capa HTTP (handlers de `/me/business...`, `/admin/...` y cableado en `main.go`) como tarea **2.7b**, antes de las vistas.
- Límite de sedes por plan: ya enforced en `AddBranch` (2.5). Los pilotos tienen límites Pro de forma TEMPORAL (`billing/adapters/pilotplan`, a borrar al terminar el piloto; luego pagan Pro). El resto queda en Starter (`fixedplan`) hasta la fase 5.
- Las imágenes del onboarding (pasos 1 y 2) contradicen el SDD en "¡Pago recibido con éxito!" y "Verificado MOVO · AUTOMÁTICO": se sigue el SDD y se consulta en la 2.8.

## Hecho
- 2.1 a 2.8 (rama `phase-2/business-onboarding`, ver `git log`). La 2.8 falta revisarla a ojo en el navegador (ver abajo).

## Pendiente
- 2.8 a 2.10 vistas (imágenes de pasos 1 y 2 ya recibidas; falta Servicios, Fotos, Sedes y Super admin → Locales).
- Variables `S3_*` en `.env.example`: bloqueado por permisos, pendiente.

## Pendiente antes o durante las vistas (detectado en 2.7b)
- `docs/openapi.yaml` (SDD §6) no existe todavía, ni para identity; decidir si se crea ahora o en la fase 8.
- `.env` debe tener `PLAN_*` y `S3_*` (más `S3_ENSURE_BUCKET=true` en desarrollo) o `cmd/api` no arranca.
- Pilotos: `billing/adapters/pilotplan` (temporal) se cablea en `cmd/api/business.go`.

## 2.8 Onboarding (hecho, pendiente de revisión visual)
- Ruta `/socios/onboarding` (cuatro pasos + pantalla "en revisión"). El portal tiene tres marcos: `(acceso)` oscuro, `onboarding` y, en la 2.9, el panel.
- El navegador llama a la API por el proxy `app/api/v1/[...path]` con lista cerrada (`me/`, `admin/`, `plans`, `categories`); `auth/*` NO pasa: el refresh token solo lo ve `app/api/session`. `API_URL` se lee en runtime (no en el build, por eso no son rewrites de next.config).
- Subida de fotos: el navegador hace PUT directo al storage con la URL prefirmada, así que el bucket necesita CORS. En desarrollo lo aplica `S3_ENSURE_BUCKET=true` + `S3_CORS_ORIGINS=http://localhost:3000`; en producción se configura en el proveedor (R2/B2/S3).
- Máximo 6 fotos (el diseño muestra "3 de 6"; el SDD no fija un número).
- Geocodificación: botón explícito "Buscar en el mapa" (SDD §12.3 prohíbe autocompletar con Nominatim público). Campo "Ciudad" agregado porque la API lo exige y el diseño no lo tenía.
- Frontend sin runner de tests: la lógica pura (`lib/business/*.ts`) se prueba con `node --test` (`npm test`, ya en la CI). Los componentes se verificaron con typecheck, lint, build y un recorrido real de la API.
- Variables que `cmd/api` exige ahora: PLAN_* (6), S3_BUCKET/ACCESS_KEY/SECRET_KEY (+ENDPOINT, PATH_STYLE, ENSURE_BUCKET, CORS_ORIGINS en dev) y GEOCODER_USER_AGENT.

## Diferencias entre los diseños y el SDD (se siguió el SDD)
- Servicios: el diseño tiene duración y precio; el modelo del SDD no. Sedes: "Piso / Depto / Timbre" tampoco existe. **Pendiente de decisión del usuario.**
- Paso 4: "Plan Trimestral / Pago online detectado / comprobante PDF" (no hay integración de pagos). Se dejó solo el número de operación opcional. Consejo con "4x más reservas" reemplazado por un consejo sin cifra inventada.
- "Navegación en un toque" (Google Maps / Waze) no existe todavía: se omitió.
- Pantalla "en revisión": sin barra de progreso "80%" ni enlace a soporte (no hay canal definido). Se conservó "Tiempo estimado < 2 hs" del diseño.
- Botón final del paso 4: "Enviar a revisión" en vez de "Siguiente", porque la acción es consecuente.
- Chips de categoría sin íconos.
