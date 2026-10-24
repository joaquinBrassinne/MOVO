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
  payment_reference: string;
  created_at: string;
}

export interface Branch {
  id: number;
  name: string;
  address: string;
  city: string;
  latitude: number;
  longitude: number;
  booking_url: string;
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
}

export interface ServiceInput {
  name: string;
  description: string;
  branch_id: number | null;
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
