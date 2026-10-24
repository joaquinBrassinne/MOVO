// Cliente tipado de los endpoints del negocio. Cada función es una llamada a la API;
// el estado y los efectos viven en los hooks.

import { authed } from "@/lib/api/authed";
import { requestPublic, type ApiResult } from "@/lib/api/request";

import type {
  BranchInput,
  Business,
  BusinessInput,
  CategoryList,
  MyBusiness,
  Place,
  PresignedPhoto,
  RegisteredPhoto,
  Service,
  ServiceInput,
  ServiceList,
  Slot,
  UploadTarget,
} from "./types";

const BASE = "/me/business";

export const listCategories = () => requestPublic<CategoryList>("/categories");

export const getMyBusiness = () => authed<MyBusiness>("GET", BASE);
export const createBusiness = (input: BusinessInput) => authed<Business>("POST", BASE, input);
export const updateBusiness = (input: BusinessInput) => authed<Business>("PUT", BASE, input);
export const submitBusiness = (paymentReference: string) =>
  authed<Business>("POST", `${BASE}/submit`, { payment_reference: paymentReference });

export const geocode = (address: string, city: string) =>
  authed<Place>("POST", `${BASE}/geocode`, { address, city });

export const addBranch = (input: BranchInput) =>
  authed<{ id: number }>("POST", `${BASE}/branches`, input);
export const editBranch = (id: number, input: BranchInput) =>
  authed<{ id: number }>("PUT", `${BASE}/branches/${id}`, input);
export const setBranchHours = (id: number, slots: Slot[]) =>
  authed<{ hours: Slot[] }>("PUT", `${BASE}/branches/${id}/hours`, { slots });

export const listServices = () => authed<ServiceList>("GET", `${BASE}/services`);
export const addService = (input: ServiceInput) =>
  authed<Service>("POST", `${BASE}/services`, input);
export const editService = (id: number, input: ServiceInput) =>
  authed<Service>("PUT", `${BASE}/services/${id}`, input);
export const removeService = (id: number) => authed<void>("DELETE", `${BASE}/services/${id}`);
export const setServiceActive = (id: number, active: boolean) =>
  authed<Service>("PUT", `${BASE}/services/${id}/active`, { active });
export const setServiceSchedule = (id: number, slots: Slot[]) =>
  authed<{ schedule: Slot[] }>("PUT", `${BASE}/services/${id}/schedule`, { slots });

export const presignPhoto = (contentType: string, size: number) =>
  authed<PresignedPhoto>("POST", `${BASE}/images/presign`, { content_type: contentType, size });
export const registerPhoto = (objectKey: string) =>
  authed<RegisteredPhoto>("POST", `${BASE}/images`, { object_key: objectKey });
export const reorderPhotos = (ids: number[]) =>
  authed<void>("PUT", `${BASE}/images/order`, { ids });
export const removePhoto = (id: number) => authed<void>("DELETE", `${BASE}/images/${id}`);

/**
 * Sube el archivo directo al servidor de objetos con la URL prefirmada. No lleva el
 * token de la API: la firma de la URL es la autorización, y el tamaño y el tipo van
 * firmados, así que el servidor rechaza cualquier otro cuerpo.
 */
export async function uploadFile(target: UploadTarget, file: File): Promise<ApiResult<void>> {
  try {
    const response = await fetch(target.url, {
      method: target.method,
      headers: target.headers,
      body: file,
    });
    if (response.ok) return { ok: true, data: undefined };
    return {
      ok: false,
      status: response.status,
      error: { code: "upload_failed", message: "the upload was rejected" },
    };
  } catch {
    return {
      ok: false,
      status: 0,
      error: { code: "network_error", message: "the upload could not be completed" },
    };
  }
}
