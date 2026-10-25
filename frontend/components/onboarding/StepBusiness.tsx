"use client";

import type { SubmitEvent } from "react";

import { CheckCircleIcon } from "@/components/socios/icons";
import { useCategories } from "@/lib/business/hooks";
import { MAX_DESCRIPTION_LENGTH, type Business } from "@/lib/business/types";
import { useBusinessForm } from "@/lib/business/useBusinessForm";

import { Field, Notice, TextArea, TextInput } from "./fields";
import { Shell, StepFooter } from "./Shell";

interface Props {
  business: Business | null;
  /** Se llama después de guardar, para volver a pedir los datos. */
  onSaved: () => void;
  onNext: () => void;
}

/** Paso 1: nombre, categoría, descripción y WhatsApp del negocio. */
export function StepBusiness({ business, onSaved, onNext }: Props) {
  const form = useBusinessForm(business);
  const categories = useCategories();
  const { fields, form: notice } = form.feedback;

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    if (await form.submit()) {
      onSaved();
      onNext();
    }
  }

  return (
    <Shell step={1}>
      <section className="rounded-3xl bg-white p-6 shadow-sm">
        <p className="inline-flex items-center gap-2 rounded-full bg-movo-accent/30 px-4 py-2 text-sm font-bold">
          <CheckCircleIcon className="size-4" />
          ¡Gracias por contratar MOVO!
        </p>
        <h2 className="mt-4 font-display text-3xl font-extrabold">Datos de tu negocio</h2>
        <p className="mt-2 text-base leading-relaxed text-movo-slate">
          Completá los datos de tu negocio. Confirmamos tu pago y revisamos tu perfil antes de
          publicarlo.
        </p>
      </section>

      <form
        id="step-form"
        onSubmit={submit}
        noValidate
        aria-busy={form.pending}
        className="space-y-6 rounded-3xl bg-white p-6 shadow-sm"
      >
        <Field
          id="name"
          label="Nombre del negocio"
          required
          error={fields.name}
          help="Así figurará en las búsquedas directas en el mapa y la app."
        >
          <TextInput
            id="name"
            value={form.name}
            onChange={form.setName}
            error={fields.name}
            disabled={form.pending}
            autoComplete="organization"
            maxLength={100}
          />
        </Field>

        <Field
          id="category"
          label="Categoría principal"
          required
          error={fields.category_id}
          hint={<span className="text-xs font-semibold text-movo-slate">Elegí 1 opción</span>}
        >
          {categories.status === "failed" ? (
            <Notice
              title="No pudimos cargar las categorías"
              body="Recargá la página para volver a intentar."
            />
          ) : (
            <div
              id="category"
              role="radiogroup"
              aria-label="Categoría principal"
              className="flex flex-wrap gap-2"
            >
              {categories.status === "ready"
                ? categories.data.map((category) => {
                    const selected = category.id === form.categoryId;
                    return (
                      <button
                        key={category.id}
                        type="button"
                        role="radio"
                        aria-checked={selected}
                        disabled={form.pending}
                        onClick={() => form.setCategoryId(category.id)}
                        className={`h-11 rounded-full px-5 text-sm font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent ${
                          selected ? "bg-movo-card text-movo-accent" : "bg-movo-sand text-movo-ink"
                        }`}
                      >
                        {category.name}
                      </button>
                    );
                  })
                : null}
            </div>
          )}
        </Field>

        <Field
          id="description"
          label="Descripción"
          error={fields.description}
          hint={
            <span className="text-xs font-semibold text-movo-slate">
              {form.description.length}/{MAX_DESCRIPTION_LENGTH}
            </span>
          }
          help="Sé directo y auténtico. Destacá el equipamiento y la vibra del lugar."
        >
          <TextArea
            id="description"
            value={form.description}
            onChange={form.setDescription}
            error={fields.description}
            disabled={form.pending}
            maxLength={MAX_DESCRIPTION_LENGTH}
            rows={5}
          />
        </Field>

        <Field
          id="phone"
          label="WhatsApp oficial"
          required
          error={fields.whatsapp}
          help="Acá te van a escribir los clientes para reservar turnos o hacer consultas directas, sin intermediarios."
        >
          <div className="flex gap-3">
            <span className="flex h-14 shrink-0 items-center rounded-2xl bg-movo-sand px-4 text-base font-semibold">
              AR +54
            </span>
            <TextInput
              id="phone"
              type="tel"
              inputMode="tel"
              autoComplete="tel-national"
              value={form.phone}
              onChange={form.setPhone}
              error={fields.whatsapp}
              disabled={form.pending}
              placeholder="11 5824 9912"
            />
          </div>
        </Field>

        {notice ? <Notice title={notice.title} body={notice.body} /> : null}
      </form>

      <p className="px-2 text-center text-sm text-movo-slate">
        Verificamos tu perfil nosotros, después de aprobar tu negocio.
      </p>

      <StepFooter pending={form.pending} />
    </Shell>
  );
}
