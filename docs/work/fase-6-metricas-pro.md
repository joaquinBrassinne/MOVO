# Fase 6 — Métricas Pro (nota de trabajo)

**Objetivo:** ampliar las métricas del plan Pro (SDD §4.3, tareas 6.6 a 6.17) sin duplicar lo ya hecho de la fase (6.1 a 6.5).

## Decisiones
- Permiso: capacidad `advanced_metrics` de `Entitlements`, un único chequeo en el servidor; Starter recibe 403 `plan_upgrade_required` sin datos. Pilotos usan el plan del piloto.
- Parámetro `period` = 7, 30 o 90. Un valor inválido es 422 `validation_failed` (el sobre de error del proyecto), no 400.
- Muestra mínima: menos de 20 cupones generados en el período → `insufficient_data`.
- "Vencido" se calcula en la consulta (`expired`, o `pending` con `expires_at` pasada), sin depender del worker.
- Tasa de canje por cohorte: de los cupones generados en el período, cuántos ya fueron canjeados.
- Servicios: `service_view` se registra con el beacon `POST /branches/{branch_id}/services/{service_id}/view` (6.5).
- Las tareas 6.6 y 6.7 (base de permiso e índices, y `average_ticket_ars`) van antes que la conversión (6.8) porque todas las métricas Pro dependen de ellas.

## Supuestos a confirmar al revisar
- Franjas del mapa de calor: madrugada 00:00–05:59, mañana 06:00–11:59, tarde 12:00–17:59, noche 18:00–23:59.
- Proyección del cupo: necesita al menos 3 días transcurridos del mes y algún cupón generado; si no, `insufficient_data`.
- "Clientes que volvieron" cuenta canjes dentro del período pedido.
- Tiempo hasta el canje: "mismo día" es el mismo día calendario de Argentina; "hasta 24 h" y "hasta 48 h" son los canjes restantes por tiempo transcurrido.

## Hecho
- 6.1 a 6.5 (ver SDD).

## Pendiente
- 6.6 a 6.20 en el orden del SDD. El backend termina en 6.19; las vistas (6.20) esperan las capturas en `docs/design/fase-6/`.
