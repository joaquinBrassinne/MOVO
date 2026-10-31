# Fase 5 — Suscripción con Mercado Pago

Rama: `phase-5/billing` (desde `origin/develop`). Estado general: `docs/sdd.md`.

## Decisiones (acordadas con el dueño del proyecto)
- **Confirmar pago es atómico:** activa el negocio (`pending_review → active`) y crea la `subscription` en una sola transacción. `approve` sigue existiendo para pilotos.
- **Gracia fija en 7 días** (sin `SUBSCRIPTION_GRACE_DAYS`): vive en la vista `visible_branches` y en `billing/domain.GraceDays`; `TestGraceDaysAgreeWithTheDomain` verifica que coinciden.
- **Downgrade con aviso:** tabla `alerts` + alerta `plan_downgrade`; el dueño la lee en `GET /me/alerts` y la descarta con `POST /me/alerts/{id}/read`.

## Supuestos (no consultados, fáciles de revertir)
- "Suspender" = la acción `pause` que ya existía (oculta al instante, sin gracia).
- `featured` de la vista = suscripción **Pro pagada** (un piloto no es destacado).
- Registrar pago: +1 mes desde `paid_until` si sigue vigente, desde hoy si ya venció (fin de mes con clamp).
- Un piloto no se recorta al bajar el plan de su suscripción (vive con límites Pro).
- Sin suscripción, los límites son los de Starter (el onboarding ocurre antes de confirmar el pago).

## Hecho (5.1–5.7, backend)
Migraciones 00022 (subscriptions), 00023 (vista), 00024 (alerts). Endpoints: `GET /plans` (ahora con `checkout_url`), `GET/POST /admin/subscriptions[/{businessId}]`, `POST …/payments`, `PUT …/plan`, `GET /admin/subscriptions?status=active|grace|expired|expiring`, `GET /me/alerts`, `POST /me/alerts/{id}/read`. Env nuevas obligatorias: `MP_STARTER_URL`, `MP_PRO_URL` (https).

## Pendiente
- **5.8 Frontend** (necesita las imágenes de las vistas: Landing con planes, Plan y suscripción, Super admin → Suscripciones).
- Los specs E2E que aprueban negocios por la UI (`frontend/e2e/03-admin`, `06-search`) dejan de verlos públicos si el negocio no es piloto ni tiene suscripción: ajustarlos en 5.8 usando "Confirmar pago".
