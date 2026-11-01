# Fase 6 — Métricas Pro (nota de trabajo)

**Objetivo:** ampliar las métricas del plan Pro (SDD §4.3, tareas 6.6 a 6.17) sin duplicar lo ya hecho de la fase (6.1 a 6.5).

## Decisiones
- Permiso: capacidad `advanced_metrics` de `Entitlements` (`PLAN_STARTER_ADVANCED_METRICS` / `PLAN_PRO_ADVANCED_METRICS`, por defecto false y true). Un único `ProGate` en `benefits` la comprueba antes de leer cualquier dato; Starter recibe 403 `plan_upgrade_required` sin datos. Un piloto usa el plan Pro que ya le da `pilotplan`.
- Parámetro `period` = 7, 30 o 90. Un valor inválido es 422 `validation_failed` (el sobre de error del proyecto), no 400. La comprobación del plan va primero.
- Muestra mínima: menos de 20 cupones generados en el período → `insufficient_data`. Un período anterior con menos de 20 cupones no es base de comparación (`previous` y `change` nulos).
- "Vencido" se calcula en la consulta (`expired`, o `pending` con `expires_at` pasada), sin depender del worker.
- Tasa de canje por cohorte: de los cupones generados en el período, cuántos ya fueron canjeados.
- Servicios: `service_view` se registra con el beacon `POST /branches/{branch_id}/services/{service_id}/view` (6.5).
- Las tareas 6.6 y 6.7 (base de permiso e índices, y `average_ticket_ars`) van antes que la conversión (6.8) porque todas las métricas Pro dependen de ellas.
- El texto de "cómo se calcula" el retorno estimado lo pone la vista (6.20): la API manda `estimated: true`, el ticket, los canjes y el precio del plan.

## Supuestos a confirmar al revisar
- Franjas del mapa de calor: madrugada 00:00–05:59, mañana 06:00–11:59, tarde 12:00–17:59, noche 18:00–23:59.
- Proyección del cupo: necesita al menos 3 días transcurridos del mes y algún cupón generado; si no, `insufficient_data`. Hoy cuenta como día transcurrido.
- "Clientes que volvieron" cuenta canjes dentro del período pedido.
- Tiempo hasta el canje: "mismo día" es el mismo día calendario de Argentina; "hasta 24 h" y "hasta 48 h" son los canjes restantes por tiempo transcurrido.
- Reseñas: el promedio cuenta las reseñas no ocultas escritas en el período (las reportadas cuentan); el "porcentaje que deja reseña" es la parte de los canjes del período con una reseña no oculta.
- Tus récords: el día y la franja fuertes y el beneficio estrella se calculan sobre los mismos 12 meses completos del ranking (no sobre un período aparte). Un mes completo de historia es un mes cerrado que empieza en o después del primer cupón del negocio.
- Comparativa por sede: lista las sedes activas aunque no tengan actividad y las retiradas solo si la tuvieron.

## Hecho
- 6.1 a 6.17 (ver SDD): backend completo con tests, OpenAPI y SDD al día.

## Pendiente
- 6.18 (alerta "Cupones sin canjear") y 6.19 (marca de prioridad en la bandeja del admin): backend de la fase fuera de este pedido.
- 6.20: en curso (ver abajo). Backend 6.18 y 6.19 hecho (commits ba65960, ba49a53, cefd6f2).

## 6.20 Frontend (especificación)
Capturas en `C:\Users\joaqu\dev\MOVO\docs\design\fase-6\` (checkout principal): `metricas_plan_starter.png`, `inteligencia_comercial_metricas_pro.png`, `datos_del_negocio_plan_pro.png`, `datos_del_negocio_plan_starter.png`, `super_admin_metricas.png`, `super_admin_suscripciones.png`, `super_admin_reseteo_de_clave.png`, `nueva_contraseña.png`.

Reglas: el SDD gana sobre la imagen; todo dato sale de la API real (`backend/api/openapi.yaml`), nunca inventado; se rotula "vistas", nunca "personas"; textos de interfaz en español, identificadores y comentarios en inglés; paths en inglés.

No se implementa (el SDD o la API no lo respaldan; se informa al cerrar): exportar CSV, «Conciliar MP», gráfico «Cupones por día», top de locales, distribución por categoría y «Crecimiento acumulado» más allá de lo que devuelve `GET /admin/metrics`, 2FA, historial de auditoría de la ficha, las barras de maqueta («Modo de vista», «Simulador», pestañas «1. Válido / 2. Éxito…»), sparklines (la API no manda series), «Sincronizado hoy», número de teléfono con bandera.

Unidades (superficies disjuntas): A métricas del dueño; B ticket promedio y alerta en el panel; C1 admin métricas y suscripciones; C2 reseteo asistido (admin) y página `/owners/reset-password`; D beacon de vista de servicio (inline).
