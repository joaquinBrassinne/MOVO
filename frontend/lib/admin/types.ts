// Contrato de los endpoints /admin/businesses de la API Go.

import type { Branch, Business, BusinessStatus } from "@/lib/business/types";

export interface AdminBusiness extends Business {
  owner_email: string;
  category_name: string;
  branch_count: number;
}

/** `counts` es cuántos negocios tiene cada estado en todo el directorio, sin importar los filtros. */
export interface AdminList {
  businesses: AdminBusiness[];
  total: number;
  counts: Partial<Record<BusinessStatus, number>>;
}

export interface AuditEntry {
  actor_email: string;
  action: string;
  at: string;
  metadata: Record<string, unknown>;
}

export interface AdminDetail {
  business: AdminBusiness;
  branches: Branch[];
  services_total: number;
  services_active: number;
  history: AuditEntry[];
}

export const ADMIN_ACTION = {
  APPROVE: "approve",
  PAUSE: "pause",
  REACTIVATE: "reactivate",
  DELETE: "delete",
  VERIFY: "verify",
  REVOKE_VERIFICATION: "revoke_verification",
  MARK_PILOT: "mark_pilot",
  UNMARK_PILOT: "unmark_pilot",
} as const;

export type AdminAction = (typeof ADMIN_ACTION)[keyof typeof ADMIN_ACTION];

export interface DirectoryFilters {
  /** Vacío es "todos menos los dados de baja". */
  status: BusinessStatus | "";
  query: string;
  /** 0 es "todas las categorías". */
  categoryId: number;
  /** Desde 1. */
  page: number;
}

export const PAGE_SIZE = 10;
