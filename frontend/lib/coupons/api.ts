// Cliente tipado de los endpoints públicos de cupones.

import { requestAnonymous } from "@/lib/api/request";

import type { ContactInput, GeneratedCoupon, HeldCoupon } from "./types";

export const generateCoupon = (benefitId: number, contact: ContactInput) =>
  requestAnonymous<GeneratedCoupon>("POST", `/benefits/${benefitId}/codes`, contact);

export const getHeldCoupon = (token: string) =>
  requestAnonymous<HeldCoupon>("GET", `/access/${encodeURIComponent(token)}`);
