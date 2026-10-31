// Pure helpers for the plan view: no React, no timezone-dependent Date parsing.

import { PLAN, type PlanName } from "../business/types.ts";

const MONTHS = [
  "enero",
  "febrero",
  "marzo",
  "abril",
  "mayo",
  "junio",
  "julio",
  "agosto",
  "septiembre",
  "octubre",
  "noviembre",
  "diciembre",
];

/** "2026-11-30" -> "30 de noviembre". Parsed by parts so UTC-3 never shifts the day. */
export function formatLongDate(isoDate: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(isoDate);
  if (!match) return isoDate;
  const month = MONTHS[Number(match[2]) - 1];
  if (!month) return isoDate;
  return `${Number(match[3])} de ${month}`;
}

export function planLabel(plan: PlanName): string {
  return plan === PLAN.PRO ? "Pro" : "Starter";
}

export interface PlanLimits {
  max_branches: number | null;
  max_active_services: number | null;
}

const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

/** The "Qué incluye tu plan" bullets; limits come from the API, null means unlimited. */
export function planFeatures(plan: PlanName, limits: PlanLimits): string[] {
  const services =
    limits.max_active_services === null
      ? "Servicios y horarios ilimitados"
      : `Hasta ${plural(limits.max_active_services, "servicio activo", "servicios activos")}`;
  const branches =
    limits.max_branches === null
      ? "Sedes ilimitadas"
      : `Hasta ${plural(limits.max_branches, "sede", "sedes")}`;

  return [
    "Perfil verificado",
    services,
    branches,
    "Botón de WhatsApp para reservar clases",
    "Ubicación en el mapa",
    "Cupones para tus clientes",
    "Reseñas y puntuación",
    ...(plan === PLAN.PRO
      ? [
          "Visibilidad destacada: prioridad en el inicio, búsquedas y pin destacado en el mapa",
          "Métricas de inteligencia comercial",
          "Soporte prioritario",
        ]
      : ["Métricas esenciales", "Soporte estándar"]),
  ];
}

/** "Norte", "Norte y Sur", "Norte, Sur y Centro". */
export function joinNames(names: string[]): string {
  if (names.length <= 1) return names.join("");
  return `${names.slice(0, -1).join(", ")} y ${names[names.length - 1]}`;
}
