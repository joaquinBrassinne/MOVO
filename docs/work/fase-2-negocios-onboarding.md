# Fase 2 — Negocios, servicios y onboarding

Rama: `phase-2/business-onboarding`. Estado general: `docs/sdd.md` §9 (la primera tarea sin tildar es por donde se sigue).

## Objetivo
Que un local pueda existir, completar su perfil y ser aprobado manualmente (sin cobro).

## Decisiones
- Un módulo no importa internals de otro, tampoco en tests (los tests de `business` crean usuarios con SQL directo).
- La descripción del negocio admite hasta 1000 caracteres; la imagen del onboarding muestra 300. Pendiente de confirmar.
- El slug se deriva del nombre y se sufija (`-2`, `-3`...) si está tomado; no cambia al editar el nombre.
- `Storage` es un puerto compartido en `internal/platform/storage`; el SDK es `aws-sdk-go-v2` (decidido por el usuario). Solo JPEG/PNG/WebP, máximo 5 MiB, el tamaño va firmado en la URL.
- Se suma la capa HTTP (handlers de `/me/business...`, `/admin/...` y cableado en `main.go`) como tarea **2.7b**, antes de las vistas.
- Límite de sedes por plan: ya enforced en `AddBranch` (2.5). Los pilotos tienen límites Pro de forma TEMPORAL (`billing/adapters/pilotplan`, a borrar al terminar el piloto; luego pagan Pro). El resto queda en Starter (`fixedplan`) hasta la fase 5.
- Las imágenes del onboarding (pasos 1 y 2) contradicen el SDD en "¡Pago recibido con éxito!" y "Verificado MOVO · AUTOMÁTICO": se sigue el SDD y se consulta en la 2.8.

## Hecho
- 2.1 a 2.6 (rama `phase-2/business-onboarding`, ver `git log`).

## Pendiente
- 2.7 super admin + `audit_log`.
- 2.7b capa HTTP y wiring.
- 2.8 a 2.10 vistas (imágenes de pasos 1 y 2 ya recibidas; falta Servicios, Fotos, Sedes y Super admin → Locales).
- Variables `S3_*` en `.env.example`: bloqueado por permisos, pendiente.
