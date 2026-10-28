// Cliente tipado de los endpoints de beneficios y cupones. Cada función es una llamada a
// la API; el estado y los efectos viven en los hooks.

import { authed } from "@/lib/api/authed";

import type {
  Activity,
  Benefit,
  BenefitInput,
  BenefitList,
  CouponPreview,
  CouponQuota,
  Overview,
  RedeemedCoupon,
} from "./types";

const BASE = "/me/business";

export const listBenefits = () => authed<BenefitList>("GET", `${BASE}/benefits`);
export const addBenefit = (input: BenefitInput) =>
  authed<Benefit>("POST", `${BASE}/benefits`, input);
export const editBenefit = (id: number, input: BenefitInput) =>
  authed<Benefit>("PUT", `${BASE}/benefits/${id}`, input);
export const setBenefitActive = (id: number, active: boolean) =>
  authed<Benefit>("PUT", `${BASE}/benefits/${id}/active`, { active });

export const getCouponQuota = () => authed<CouponQuota>("GET", `${BASE}/coupon-quota`);
/** `null` saca el límite. */
export const setCouponQuota = (limit: number | null) =>
  authed<CouponQuota>("PUT", `${BASE}/coupon-quota`, { limit });

export const getOverview = () => authed<Overview>("GET", `${BASE}/overview`);
/** `since` es la marca (RFC 3339) del último hecho que el panel ya mostró. */
export const getActivity = (since?: string) =>
  authed<Activity>(
    "GET",
    since ? `${BASE}/activity?since=${encodeURIComponent(since)}` : `${BASE}/activity`,
  );

export const previewCoupon = (code: string) =>
  authed<CouponPreview>("GET", `${BASE}/codes/${encodeURIComponent(code)}`);
/** `branchId` solo hace falta cuando el negocio tiene más de una sede activa. */
export const redeemCoupon = (code: string, branchId?: number) =>
  authed<RedeemedCoupon>("POST", `${BASE}/codes/redeem`, {
    code,
    ...(branchId === undefined ? {} : { branch_id: branchId }),
  });
