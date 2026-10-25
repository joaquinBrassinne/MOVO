"use client";

import { useState } from "react";

import { addService, editService, setServiceSchedule } from "./api";
import { describeFailure, type Feedback } from "./messages";
import { buildSlots, findOverlap, isValidRange, toRanges, type Range } from "./schedule";
import { parsePrice } from "./price";
import type { Service } from "./types";

const EMPTY_DRAFT: Range = { start: "", end: "" };

/** Estado y envío del formulario de un servicio (alta o edición) con su horario semanal. */
export function useServiceForm(existing: Service | null) {
  // Un horario que cambia entre días no cabe en "estos días, estas franjas": se deja como está.
  const weekly = existing ? toRanges(existing.schedule) : { days: [], ranges: [] };
  const scheduleEditable = weekly !== null;

  const [name, setName] = useState(existing?.name ?? "");
  const [description, setDescription] = useState(existing?.description ?? "");
  const [price, setPrice] = useState(existing?.price ? String(existing.price) : "");
  const [days, setDays] = useState<number[]>(weekly?.days ?? []);
  const [ranges, setRanges] = useState<Range[]>(weekly?.ranges ?? []);
  const [draft, setDraft] = useState<Range>(EMPTY_DRAFT);
  const [drafting, setDrafting] = useState(false);
  const [pending, setPending] = useState(false);
  const [feedback, setFeedback] = useState<Feedback>({ fields: {} });

  function toggleDay(weekday: number) {
    setDays((current) =>
      current.includes(weekday) ? current.filter((d) => d !== weekday) : [...current, weekday],
    );
  }

  /** Suma la franja que se está escribiendo, si es válida y no pisa a otra. */
  function commitDraft() {
    const next = [...ranges, draft];
    if (!isValidRange(draft)) {
      setFeedback({ fields: { slots: "Elegí una hora de inicio y otra de fin posterior" } });
      return;
    }
    if (findOverlap(next)) {
      setFeedback({ fields: { slots: "Esa franja se pisa con otra" } });
      return;
    }

    setRanges(next.sort((a, b) => a.start.localeCompare(b.start)));
    setDraft(EMPTY_DRAFT);
    setDrafting(false);
    setFeedback({ fields: {} });
  }

  function removeRange(index: number) {
    setRanges((current) => current.filter((_, i) => i !== index));
  }

  /** Valida, guarda el servicio y su horario, y dice si se pudo. */
  async function submit(): Promise<boolean> {
    if (pending) return false;

    const fields: Record<string, string> = {};
    if (name.trim() === "") fields.name = "Ingresá el nombre del servicio";
    const amount = parsePrice(price);
    if (amount === undefined) fields.price = "Ingresá un precio válido en pesos";
    if (scheduleEditable && ranges.length > 0 && days.length === 0) {
      fields.days = "Elegí al menos un día";
    }
    if (Object.keys(fields).length > 0) {
      setFeedback({ fields });
      return false;
    }

    setPending(true);
    setFeedback({ fields: {} });

    const input = {
      name,
      description,
      branch_id: existing?.branch_id ?? null,
      price: amount ?? null,
    };
    const saved = await (existing ? editService(existing.id, input) : addService(input));
    if (!saved.ok) {
      setPending(false);
      setFeedback(describeFailure(saved));
      return false;
    }

    if (scheduleEditable) {
      const slots = buildSlots(days, ranges);
      const scheduled = await setServiceSchedule(saved.data.id, slots);
      if (!scheduled.ok) {
        setPending(false);
        setFeedback(describeFailure(scheduled));
        return false;
      }
    }

    setPending(false);
    return true;
  }

  return {
    name,
    setName,
    description,
    setDescription,
    price,
    setPrice,
    days,
    toggleDay,
    ranges,
    draft,
    setDraft,
    drafting,
    startDraft: () => setDrafting(true),
    commitDraft,
    removeRange,
    scheduleEditable,
    pending,
    feedback,
    submit,
  };
}
