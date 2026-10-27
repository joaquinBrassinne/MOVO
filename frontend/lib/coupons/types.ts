// Contrato de los endpoints del cliente, que no tiene cuenta: generar un cupón y ver el que
// tiene por su enlace de acceso.

import type { ClaimedBenefit, CouponStatus } from "../benefits/types.ts";

/** Lo que devuelve generar un cupón. El token de acceso aparece solo acá: el servidor guarda su hash. */
export interface GeneratedCoupon {
  code: string;
  access_token: string;
  expires_at: string;
  benefit: ClaimedBenefit;
}

/** El cupón que abre un enlace de acceso. Nunca trae el teléfono ni el token. */
export interface HeldCoupon {
  code: string;
  status: CouponStatus;
  business_name: string;
  expires_at: string;
  redeemed_at: string | null;
  benefit: ClaimedBenefit;
}

export interface ContactInput {
  name: string;
  /** El WhatsApp en formato wa.me (54 9 + 10 dígitos). */
  phone: string;
}
