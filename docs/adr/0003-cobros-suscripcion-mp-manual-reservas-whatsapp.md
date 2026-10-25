# 0003 — Cobros por plan de suscripción de Mercado Pago con registro manual y reservas por WhatsApp en el MVP

- **Estado:** aceptada
- **Fecha:** 2026-10-24
- **Decide sobre:** SDD §1.2, §1.5, §3.3, §5.3, §5.4, §9 (Fases 3 y 5)

## Contexto

Para el piloto inicial en Almafuerte y Río Tercero (5 a 10 locales activos), la prioridad es validar la propuesta de valor sin asumir complejidad operacional ni costos de integración prematuros:
1. **Cobros recurrentes a locales:** Integrar la API de Mercado Pago (OAuth, webhooks, firmas, conciliación, gestión de tarjetas y reintentos) requiere un esfuerzo técnico significativo y mantenimiento continuo de endpoints expuestos.
2. **Reservas de clases y turnos:** Implementar un sistema de turnos, agendas, calendarios, cupos y estados de reserva ("pendiente", "confirmada", "cancelada") agrega sobrecarga transaccional y fricción a un usuario que busca inmediatez.

## Decisión

1. **Planes de suscripción de Mercado Pago sin integración de API:**
   - Se crean planes de suscripción preaprobados en la plataforma de Mercado Pago (links configurados por variables de entorno `MP_STARTER_URL` y `MP_PRO_URL`).
   - El dueño del local abre el enlace una sola vez y autoriza el débito automático mensual que gestiona Mercado Pago.
   - Mercado Pago redirige a `/onboarding` donde el dueño completa sus datos y declara el email y el nombre completo con los que pagó (`mp_payer_email`, `mp_payer_name`), porque Mercado Pago no entrega comprobante ni número de operación al pagador. Son una referencia informativa (nunca prueba de pago) y el plan no se pregunta: lo fija el super admin al confirmar, según el mail "nuevo suscriptor" que recibe de Mercado Pago.
   - **Registro manual:** El super admin valida el cobro en su panel de Mercado Pago y activa el negocio fijando `paid_until`. Cada mes, revisa cobros y ejecuta "Registrar pago" en el panel de administración de MOVO para extender la vigencia.
   - Si un pago no ingresa, opera un período de gracia de 7 días antes de ocultar el local en la búsqueda.
2. **Reservas y contacto delegados a WhatsApp:**
   - Sin reservas con estados ni agendas en MOVO: el contacto se realiza mediante enlace directo `wa.me/<número>?text=<mensaje prearmado>`.
   - La API no expone el teléfono del dueño públicamente; se accede mediante una redirección `GET /r/wa/{branch_id}` que incrementa el contador de clics y deriva a la conversación externa.
   - La coordinación de días, horarios y disponibilidad ocurre enteramente entre el cliente y el local por WhatsApp.
3. **Cero proveedores externos:** Sin SDKs de Mercado Pago, sin Twilio/WhatsApp Business API y sin colas de mensajería externa en el MVP.

## Consecuencias

- **Positivas:**
  - Reducción drástica del alcance del MVP: sin endpoints de webhooks para cobros, sin tablas de estado de reservas ni sincronización distribuida.
  - Mayor seguridad: no se manejan credenciales sensibles de cobro ni datos bancarios en el backend de MOVO.
  - Operación viable para el tamaño del piloto: el tiempo administrativo de revisar 5 a 10 cobros mensuales es de pocos minutos al mes.
  - Conversión más rápida para los usuarios finales, sin fricción de registro ni confirmación de turnos dentro de la app.
- **Negativas / Costos:**
  - Dependencia de un proceso operativo manual mensual por parte del super admin.
  - Riesgo de error humano al registrar pagos o al cruzar el email y el nombre declarados con el mail de Mercado Pago.
  - Pérdida de trazabilidad fina dentro del sistema sobre cancelaciones o desuscripciones en Mercado Pago hasta la revisión mensual.

## Cuándo automatizar con webhooks (Trigger de transición)

La automatización de cobros vía API y webhooks de Mercado Pago se implementará **post-MVP**, cuando el volumen alcance aproximadamente **15 a 20 locales activos**, o cuando la carga operativa de conciliación manual mensual supere una hora por mes. En ese punto:
- Se implementará el endpoint de recepción de webhooks de Mercado Pago con verificación de firmas criptográficas.
- Los eventos se persistirán primero en PostgreSQL y se procesarán de forma asíncrona e idempotente en `cmd/worker`.
- Se gestionarán altas, renovaciones, bajas y fallos de débito de forma 100% automática.

## Alternativas descartadas

- **Integración completa de Mercado Pago API (Checkout Pro / Subscriptions API) en el MVP:** Agregaba semanas de desarrollo, necesidad de URLs públicas para webhooks en local y gestión de credenciales en un piloto con menos de 10 clientes.
- **Módulo de reservas interno con calendario y estados:** Fricción para los locales (que ya coordinan por WhatsApp) y riesgo de desincronización con la ocupación física real.
