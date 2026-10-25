// Contrato de los endpoints /me/business, /categories y /plans de la API Go.
// Los nombres de campo son los del JSON, por eso van en snake_case.

export const BUSINESS_STATUS = {
  DRAFT: "draft",
  PENDING_REVIEW: "pending_review",
  ACTIVE: "active",
  PAUSED: "paused",
  DELETED: "deleted",
} as const;

export type BusinessStatus = (typeof BUSINESS_STATUS)[keyof typeof BUSINESS_STATUS];

/** Límites del negocio que la interfaz necesita conocer (SDD §4.2 y la API). */
export const MAX_DESCRIPTION_LENGTH = 500;
export const MAX_PHOTOS = 6;
export const MAX_REFERENCE_LENGTH = 100;
export const MAX_PHOTO_BYTES = 5 * 1024 * 1024;
export const PHOTO_TYPES = ["image/jpeg", "image/png", "image/webp"];

export interface Category {
  id: number;
  slug: string;
  name: string;
}

/** Un tramo de apertura. `weekday` va de 0 (domingo) a 6 (sábado); la hora es "HH:MM". */
export interface Slot {
  weekday: number;
  start: string;
  end: string;
}

export interface Business {
  id: number;
  category_id: number;
  name: string;
  slug: string;
  description: string;
  whatsapp: string;
  status: BusinessStatus;
  pilot: boolean;
  verified: boolean;
  /** Email y nombre con los que el dueño dice haber pagado en Mercado Pago. */
  mp_payer_email: string;
  mp_payer_name: string;
  /** Plan que sugería la URL de retorno; solo una pista, el admin fija el plan. */
  mp_plan_hint: string;
  created_at: string;
}

/** Lo que el dueño declara al enviar el negocio a revisión. */
export interface PayerInput {
  email: string;
  name: string;
  returnParams: Record<string, string>;
}

export interface Branch {
  id: number;
  name: string;
  address: string;
  city: string;
  latitude: number;
  longitude: number;
  booking_url: string;
  reference: string;
  active: boolean;
  hours: Slot[];
}

export interface Photo {
  id: number;
  position: number;
  url: string;
}

export interface MyBusiness {
  business: Business;
  branches: Branch[];
  images: Photo[];
}

export interface Service {
  id: number;
  branch_id: number | null;
  name: string;
  description: string;
  /** Precio estimado en pesos enteros; `null` si el dueño no lo cargó. */
  price: number | null;
  active: boolean;
  schedule: Slot[];
}

export interface ServiceList {
  services: Service[];
}

export interface CategoryList {
  categories: Category[];
}

export interface BusinessInput {
  category_id: number;
  name: string;
  description: string;
  whatsapp: string;
}

export interface BranchInput {
  name: string;
  address: string;
  city: string;
  latitude: number;
  longitude: number;
  booking_url: string;
  reference: string;
}

export interface ServiceInput {
  name: string;
  description: string;
  branch_id: number | null;
  price: number | null;
}

export interface Place {
  latitude: number;
  longitude: number;
  display_name: string;
}

export interface UploadTarget {
  url: string;
  method: string;
  headers: Record<string, string>;
}

export interface PresignedPhoto {
  object_key: string;
  upload: UploadTarget;
}

export interface RegisteredPhoto {
  id: number;
  position: number;
  object_key: string;
}

export const PLAN = {
  STARTER: "starter",
  PRO: "pro",
} as const;

export type PlanName = (typeof PLAN)[keyof typeof PLAN];

/** El plan del negocio con sus límites. `null` en un límite significa sin tope. */
export interface PlanSummary {
  plan: PlanName;
  price: number;
  max_branches: number | null;
  max_active_services: number | null;
}

export interface BranchActivation {
  id: number;
  active: boolean;
}
