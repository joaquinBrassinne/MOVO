import { authed } from "@/lib/api/authed";
import type { Business } from "@/lib/business/types";

import {
  PAGE_SIZE,
  type AdminAction,
  type AdminDetail,
  type AdminList,
  type DirectoryFilters,
} from "./types";

export function listAdminBusinesses(filters: DirectoryFilters) {
  const params = new URLSearchParams({
    limit: String(PAGE_SIZE),
    offset: String((filters.page - 1) * PAGE_SIZE),
  });
  if (filters.status) params.set("status", filters.status);
  if (filters.query.trim()) params.set("q", filters.query.trim());
  if (filters.categoryId) params.set("category_id", String(filters.categoryId));

  return authed<AdminList>("GET", `/admin/businesses?${params}`);
}

export const getAdminBusiness = (id: number) =>
  authed<AdminDetail>("GET", `/admin/businesses/${id}`);

export const applyAdminAction = (id: number, action: AdminAction) =>
  authed<Business>("POST", `/admin/businesses/${id}/actions`, { action });
