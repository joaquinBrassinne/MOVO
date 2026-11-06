# Endurecimiento tras la auditoría de `develop`

Rama: `fix/audit-p1-hardening` (desde `develop` en `e86fd36`). Antes de la Fase 5.
Los hallazgos son AUD-0xx de la auditoría técnica de esa rama.

## Hecho

- AUD-002 El log ya no escribe el path: `httpx.Route` da el patrón de chi (`/access/{token}`), también en los logs de error 5xx.
- AUD-004 y AUD-018 (sedes y servicios) Los límites del plan se cuentan y se escriben bajo un lock del negocio (`CreateIfRoom`, `SetActive` con `business.Fits`). `Save` ya no escribe `active`.
- AUD-006 Cabeceras de seguridad en todas las rutas. Sin `script-src`: Next mete scripts en línea sin nonce.
- AUD-007 El registro cuenta cada intento por IP: 10 por hora, 429 con `Retry-After`.
- AUD-015 El proxy del navegador acepta solo segmentos `[A-Za-z0-9_-]`.
- AUD-003 Un cupón conserva las condiciones con las que se emitió (migración 00020: copia en `benefit_codes`, backfill desde el beneficio actual y trigger para quien inserte sin ellas).
- Historial de canjes: la vista previa del dueño lista los últimos 10 canjes del cliente en el negocio (`previous_redemptions`), con las condiciones que tenían, la sede y la fecha.

- AUD-008 (parte) argon2 corre de a 4 a la vez (`maxConcurrentHashes`); el resto de las llamadas hace cola en vez de reservar 19 MiB cada una.
- AUD-010 Borrar un servicio es una baja lógica (`deleted_at`, migración 00021): sale del panel, del perfil público y del conteo del plan, pero conserva sus métricas. La FK de `metric_daily` pasó a `RESTRICT`.
- AUD-009 El worker corre `purge-objects` (lock 4003, hasta 200 claves por vuelta) y reintenta los borrados que fallaron. Cada tarea tiene un tope de 10 minutos para no retener su lock.
- AUD-018 (beneficios) `Save` ya no escribe `active` y el interruptor es `SetActive`, un solo `UPDATE`: una edición con una copia vieja no deshace un apagado.
- AUD-017 Un teléfono es una persona y conserva el primer nombre que declaró: un cupón en otro local ya no lo renombra.

## Pendiente

- AUD-001 La IP del cliente depende de cómo se despliegue Next. Se cierra en la tarea 8.4 (Next debe agregar la IP de quien se le conecta, y la API no debe arrancar en producción con `TRUSTED_PROXIES` vacío).
- AUD-005 El downgrade no se aplica al quitar el piloto: entra con la tarea 5.7.
- AUD-008 (resto) El intento de login se cuenta recién al fallar: ráfagas en paralelo pasan el límite antes de que se registre el primer fallo.
- CSP con `script-src` (necesita nonces).
- Los e2e de Playwright no se corrieron: necesitan la pila levantada.
