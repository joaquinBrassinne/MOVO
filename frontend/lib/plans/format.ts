import type { Plan } from "./types.ts";

const pesos = new Intl.NumberFormat("es-AR", {
  style: "currency",
  currency: "ARS",
  maximumFractionDigits: 0,
});

/** Formats whole pesos the Argentine way: 19800 -> "$19.800" (no decimals, no spaces). */
export function formatPrice(amount: number): string {
  return pesos.format(amount).replace(/\s/g, "");
}

function services(limit: number | null): string {
  return limit === null
    ? "Servicios ilimitados"
    : `Hasta ${limit} ${limit === 1 ? "servicio" : "servicios"}`;
}

function branches(limit: number): string {
  return `Hasta ${limit} ${limit === 1 ? "sede" : "sedes"}`;
}

/** Feature bullets of a plan (SDD 4.2); the numbers come from the catalog, never hardcoded. */
export function planFeatures(plan: Plan): string[] {
  if (plan.plan === "pro") {
    return [
      "Todo lo de Starter",
      services(plan.max_active_services),
      branches(plan.max_branches),
      "Visibilidad destacada en el inicio, las búsquedas y el mapa",
      "Métricas de inteligencia comercial",
      "Soporte prioritario",
    ];
  }
  return [
    "Perfil verificado",
    services(plan.max_active_services),
    branches(plan.max_branches),
    "Botón de WhatsApp para reservar",
    "Ubicación en el mapa",
    "Cupones",
    "Reseñas y puntuación",
    "Métricas esenciales",
    "Soporte estándar",
  ];
}
