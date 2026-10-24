# Fase 1 — Identidad y autorización

Rama `phase-1/identity` (desde `develop`). Estado general: checklist de la Fase 1 en `docs/sdd.md`.

## Objetivo

Login seguro de dueños y super admin. Los clientes no tienen cuenta.

## Decisiones

- **IDs:** `bigint GENERATED ALWAYS AS IDENTITY` (el SDD admite "UUID v7 o bigserial"; Postgres 16 no trae `uuidv7()`). Las fases siguientes siguen la misma convención.
- **Email:** se normaliza a minúsculas y sin espacios en el dominio; `UNIQUE` sobre el valor normalizado.
- **Contraseñas:** argon2id, formato PHC. Mínimo 10 y máximo 128 caracteres.
- **Access token:** JWT HS256, `sub` = id del usuario, `role`, TTL corto. Secreto por env (`JWT_SECRET`, ≥ 32 bytes).
- **Refresh token:** valor opaco aleatorio de 256 bits, se guarda su SHA-256. Rota en cada uso (`replaced_by`); reutilizar uno ya rotado revoca toda la cadena del usuario.
- **Login:** mismo error para email inexistente y contraseña incorrecta, con hash de relleno para no filtrar por tiempo.
- **Registro público:** solo crea `business_admin`. El `super_admin` sale únicamente del seed por env.
- **Límite de intentos:** en memoria detrás de un puerto (`LoginLimiter`), contando fallos por email y por IP (la IP vive solo en memoria, no se persiste).
- **Middleware JWT y roles:** en `internal/platform/httpx` para que cualquier módulo lo use sin importar internals de `identity`.
- **Entrega de tokens:** en el cuerpo JSON de la respuesta. Dónde guardarlos en el navegador se decide en 1.7.

## Orden

1.1 → 1.2 → 1.3 (casos de uso, Postgres, JWT) → 1.4 → 1.5 → 1.3 (HTTP + composición, porque Login depende del limitador) → 1.6 → 1.7.

## Entorno local

El puerto 5432 lo ocupa otro contenedor (`locos_deporte_db`). MOVO se levanta con `POSTGRES_PORT=5433` y `DATABASE_URL=...:5433/...` exportados en el shell, sin tocar `.env`.

## Hecho

- **1.1** migración `00002_identity.sql` (`users`, `refresh_tokens`) con tests contra Postgres real (`internal/platform/dbtest`, testcontainers-go v0.44.0).
- **1.2** dominio `identity` (`User`, roles, email, política de contraseña, argon2id PHC) y reglas depguard de capas.
- **1.3** casos de uso `Register`/`Login`/`Refresh`/`Logout`, adaptadores Postgres (rotación atómica con detección de reuso), firmador JWT HS256 y endpoints `/api/v1/auth/*` compuestos en `cmd/api`. `config.LoadAuth` se carga aparte de `config.Load` para no exigir `JWT_SECRET` a `cmd/migrate`.
- **1.4** `httpx.Authenticate` (Bearer) y `httpx.RequireRole`, con el `Principal` en el contexto.
- **1.5** limitador de intentos en memoria (`adapters/ratelimit`) detrás del puerto `LoginLimiter`, con tope de claves.
- **1.6** seed del primer super admin por `SUPER_ADMIN_EMAIL`/`SUPER_ADMIN_PASSWORD` al arrancar la API: idempotente, no pisa cuentas existentes y se niega a tocar el email de un dueño.
- **1.7** vistas de ingreso y alta (`/socios/ingresar`, `/socios/crear-cuenta`), BFF de sesión (`/api/session/*`) con refresh token en cookie `httpOnly`, hook de sesión con renovación serializada y panel provisorio (`/socios/panel`). Verificado en navegador (Edge headless por CDP): 15 comprobaciones de flujo OK.

## Pendiente

- Cierre de fase (DoD) y decisiones del usuario listadas abajo.

## Antes de desplegar detrás de un proxy

`clientIP` usa la dirección del par TCP. Detrás de un reverse proxy todos los clientes comparten una IP y el límite por dirección bloquearía a todos tras 5 fallos. Hay que resolver la IP real con proxies de confianza (tarea de despliegue, Fase 8).

## Diseño de la 1.7 contra el SDD (decisiones a confirmar)

Donde el diseño contradice el SDD se siguió el SDD:

- "¿Olvidaste tu contraseña?" y "restablecé tu clave": el SDD §5.7 define reseteo **asistido por WhatsApp** (sin email). El link abre WhatsApp de soporte con `NEXT_PUBLIC_SUPPORT_WHATSAPP`; sin número se oculta. El texto del error se ajustó.
- "Cifrado bancario B2B" y "Liquidaciones 24h": el MVP no procesa pagos ni liquida (SDD §1.2). Se omitieron.
- "Sin costo de alta ni comisiones fijas. Activá tus pases…": los locales pagan suscripción mensual y el término es "cupón". Se omitió.
- "Términos de Negocios MOVO": los términos llegan en la tarea 8.5; no se enlaza una página que no existe.
- Contraseña de 8+ caracteres en el diseño; el backend exige 10 (decisión de la 1.2). La UI muestra 10.

Pendientes de diseño: logotipo con la onda (hoy es texto, falta el archivo), tokens de color estimados desde la imagen, y el ícono de error del ingreso va contorneado en lugar de relleno.

## Para fases siguientes

- `sqlc` todavía no está en el repo: los adaptadores usan `pgx` directo.
- No hay runner de tests en el frontend (`package.json` no tiene uno); la lógica pura (`lib/session/messages.ts`) quedó lista para probarse.
- Las llamadas autenticadas de la Fase 2 necesitan un wrapper que renueve el access token ante un 401 (hoy nada lo usa).
