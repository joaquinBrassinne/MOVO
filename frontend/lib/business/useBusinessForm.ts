"use client";

import { useState } from "react";

import { createBusiness, updateBusiness } from "./api";
import { useOptionalBusiness } from "./BusinessProvider";
import { describeFailure, type Feedback } from "./messages";
import { fromWhatsApp, toWhatsApp } from "./phone";
import { parseTicket } from "./price";
import { BUSINESS_STATUS, MAX_DESCRIPTION_LENGTH, type Business } from "./types";

/**
 * Estado y envío del paso 1: los datos del negocio. El ticket promedio solo se ofrece sobre un
 * negocio ya creado (no en borrador) y solo se edita si el plan tiene métricas avanzadas.
 */
export function useBusinessForm(existing: Business | null) {
  const plan = useOptionalBusiness()?.plan;
  const [ticket, setTicket] = useState(
    existing?.average_ticket_ars ? String(existing.average_ticket_ars) : "",
  );
  const ticketAvailable = existing !== null && existing.status !== BUSINESS_STATUS.DRAFT;
  const ticketEditable = ticketAvailable && plan?.status === "ready" && plan.data.advanced_metrics;
  // Hasta saber el plan no se muestra nada: evita un parpadeo de la tarjeta bloqueada.
  const ticketLocked = ticketAvailable && plan?.status === "ready" && !plan.data.advanced_metrics;
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
    const averageTicket = ticketEditable ? parseTicket(ticket) : null;
    if (averageTicket === undefined) {
      fields.average_ticket_ars = "Ingresá el monto en pesos enteros, mayor que cero";
    }
    if (Object.keys(fields).length > 0 || categoryId === null || whatsapp === null) {
      setFeedback({ fields });
      return false;
    }

    setPending(true);
    setFeedback({ fields: {} });

    const input = {
      category_id: categoryId,
      name,
      description,
      whatsapp,
      // Ausente conserva lo guardado: un plan sin métricas avanzadas no puede tocarlo.
      ...(ticketEditable && existing ? { average_ticket_ars: averageTicket } : {}),
    };
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
    ticket,
    setTicket,
    ticketEditable,
    ticketLocked,
    pending,
    feedback,
    submit,
  };
}
