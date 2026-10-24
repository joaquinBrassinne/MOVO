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

## Pendiente

1.1 a 1.7.
