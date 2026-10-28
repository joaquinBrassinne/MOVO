// Contrato de los endpoints de beneficios y cupones de la API Go (SDD §5.2).
// Los nombres de campo son los del JSON, por eso van en snake_case.

export const BENEFIT_TYPE = {
  PERCENT: "percent",
  AMOUNT: "amount",
  CUSTOM: "custom",
} as const;

export type BenefitType = (typeof BENEFIT_TYPE)[keyof typeof BENEFIT_TYPE];

/** Límites que la interfaz muestra; la API los valida igual. */
export const MAX_TITLE_LENGTH = 80;
export const MAX_CONDITIONS_LENGTH = 120;
export const MAX_PERCENT = 100;
export const MAX_AMOUNT_ARS = 100_000_000;
/** Atajos de la interfaz: no son un límite, el porcentaje es cualquier entero de 1 a 100. */
export const PERCENT_SHORTCUTS = [10, 15, 20] as const;
export const QUOTA_SHORTCUTS = [20, 30, 50, 100] as const;

/** Un beneficio como lo ve un cliente: lo que vale y bajo qué condiciones. */
export interface ClaimedBenefit {
  id: number;
  type: BenefitType;
  percent: number | null;
  amount_ars: number | null;
  title: string | null;
  conditions: string | null;
  /** Lo que muestran tarjetas y pills: "-15%", "-$2.000" o "Beneficio". */
  short_label: string;
}

/** Un beneficio como lo lee el panel del dueño. */
export interface Benefit extends ClaimedBenefit {
  /** El interruptor del dueño. */
  active: boolean;
  /** La pausa del super admin: el dueño la ve, pero no la puede cambiar. */
  admin_paused: boolean;
}

export interface BenefitListItem extends Benefit {
  generated_this_month: number;
  redeemed_this_month: number;
}

export interface BenefitList {
  benefits: BenefitListItem[];
}

/** Lo que se envía al crear o editar: solo el campo del tipo elegido y las condiciones. */
export interface BenefitInput {
  type: BenefitType;
  percent?: number;
  amount_ars?: number;
  title?: string;
  conditions?: string;
}

/** El cupo mensual del local. `limit` es `null` cuando no tiene límite. */
export interface CouponQuota {
  limit: number | null;
  generated_this_month: number;
  /** Fecha (AAAA-MM-DD, hora de Argentina) en que el conteo vuelve a empezar. */
  resets_on: string;
}

/** Los números del inicio del panel. Las vistas y los clics son de los últimos 7 días. */
export interface Overview {
  active_coupons: number;
  redeemed_today: number;
  profile_views_7d: number;
  whatsapp_clicks_7d: number;
}

export const ACTIVITY_TYPE = {
  GENERATED: "coupon_generated",
  REDEEMED: "coupon_redeemed",
} as const;

export type ActivityType = (typeof ACTIVITY_TYPE)[keyof typeof ACTIVITY_TYPE];

export interface ActivityItem {
  type: ActivityType;
  /** Momento del hecho, en RFC 3339. */
  at: string;
  customer_name: string;
  benefit: ClaimedBenefit;
}

export interface Activity {
  items: ActivityItem[];
  /** Cuántos hechos son posteriores a la marca `since` que se envió. */
  new_count: number;
}

export const COUPON_STATUS = {
  PENDING: "pending",
  REDEEMED: "redeemed",
  EXPIRED: "expired",
} as const;

export type CouponStatus = (typeof COUPON_STATUS)[keyof typeof COUPON_STATUS];

/** El cupón de un negocio antes de canjearlo. Nunca trae el teléfono del cliente. */
export interface CouponPreview {
  code: string;
  status: CouponStatus;
  customer_name: string;
  customer_redeemed_count: number;
  expires_at: string;
  redeemed_at: string | null;
  benefit: ClaimedBenefit;
}

export interface RedeemedCoupon {
  code: string;
  redeemed_at: string;
  customer_name: string;
  branch_id: number;
  benefit: ClaimedBenefit;
}
