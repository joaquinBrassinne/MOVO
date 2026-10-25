"use client";

import { useState } from "react";

import { createBusiness, updateBusiness } from "./api";
import { describeFailure, type Feedback } from "./messages";
import { fromWhatsApp, toWhatsApp } from "./phone";
import { MAX_DESCRIPTION_LENGTH, type Business } from "./types";

/** Estado y envío del paso 1: los datos del negocio. */
export function useBusinessForm(existing: Business | null) {
  const [name, setName] = useState(existing?.name ?? "");
  const [categoryId, setCategoryId] = useState<number | null>(existing?.category_id ?? null);
  const [description, setDescription] = useState(existing?.description ?? "");
  const [phone, setPhone] = useState(existing ? fromWhatsApp(existing.whatsapp) : "");
  const [pending, setPending] = useState(false);
  const [feedback, setFeedback] = useState<Feedback>({ fields: {} });

  /** Valida, guarda (crea o actualiza) y dice si se pudo. Los errores quedan en `feedback`. */
  async function submit(): Promise<boolean> {
    if (pending) return false;

    const whatsapp = toWhatsApp(phone);
    const fields: Record<string, string> = {};
    if (name.trim() === "") fields.name = "Ingresá el nombre de tu negocio";
    if (categoryId === null) fields.category_id = "Elegí una categoría";
    if (description.length > MAX_DESCRIPTION_LENGTH) {
      fields.description = `Máximo ${MAX_DESCRIPTION_LENGTH} caracteres`;
    }
    if (whatsapp === null) fields.whatsapp = "Ingresá tu número con característica, sin 0 ni 15";
    if (Object.keys(fields).length > 0 || categoryId === null || whatsapp === null) {
      setFeedback({ fields });
      return false;
    }

    setPending(true);
    setFeedback({ fields: {} });

    const input = { category_id: categoryId, name, description, whatsapp };
    const result = await (existing ? updateBusiness(input) : createBusiness(input));
    setPending(false);

    if (result.ok) return true;
    setFeedback(describeFailure(result));
    return false;
  }

  return {
    name,
    setName,
    categoryId,
    setCategoryId,
    description,
    setDescription,
    phone,
    setPhone,
    pending,
    feedback,
    submit,
  };
}
