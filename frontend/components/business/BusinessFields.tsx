"use client";

import { Field, Notice, TextArea, TextInput } from "@/components/onboarding/fields";
import { useCategories } from "@/lib/business/hooks";
import { MAX_DESCRIPTION_LENGTH } from "@/lib/business/types";
import type { useBusinessForm } from "@/lib/business/useBusinessForm";

interface Props {
  form: ReturnType<typeof useBusinessForm>;
}

/**
 * Los campos de los datos del negocio: nombre, categoría, descripción y WhatsApp. Los usan el
 * alta (paso 1 del onboarding) y la pantalla "Datos del negocio" del panel.
 */
export function BusinessFields({ form }: Props) {
  const categories = useCategories();
  const { fields, form: notice } = form.feedback;

  return (
    <>
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
    </>
  );
}
