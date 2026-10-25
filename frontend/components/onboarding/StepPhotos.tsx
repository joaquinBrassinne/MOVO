"use client";

import { useState, type SubmitEvent } from "react";

import { CameraIcon, ImageIcon, StarIcon, XIcon } from "@/components/socios/icons";
import { submitBusiness } from "@/lib/business/api";
import { describeFailure, type FormMessage } from "@/lib/business/messages";
import { MAX_PHOTOS, PHOTO_TYPES, type Business, type Photo } from "@/lib/business/types";
import { usePhotos } from "@/lib/business/usePhotos";

import { Field, Notice, TextInput } from "./fields";
import { Shell, StepFooter } from "./Shell";

interface Props {
  business: Business;
  photos: Photo[];
  /** Se llama después de cambiar las fotos o de enviar el negocio, para volver a pedir los datos. */
  onChanged: () => void;
  onBack: () => void;
}

/** Paso 4: fotos del local y número de operación; al terminar el negocio pasa a revisión. */
export function StepPhotos({ business, photos, onChanged, onBack }: Props) {
  const gallery = usePhotos(photos, onChanged);
  const [reference, setReference] = useState(business.payment_reference);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState<FormMessage | null>(null);
  const [referenceError, setReferenceError] = useState<string | undefined>();

  const [cover, ...others] = photos;

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending || gallery.busy) return;

    setPending(true);
    setMessage(null);
    setReferenceError(undefined);

    const result = await submitBusiness(reference);
    setPending(false);

    if (result.ok) {
      onChanged();
      return;
    }
    const feedback = describeFailure(result);
    setReferenceError(feedback.fields.payment_reference);
    setMessage(feedback.form ?? null);
  }

  return (
    <Shell step={4}>
      <section>
        <h2 className="font-display text-4xl font-extrabold">Fotos y pago</h2>
        <p className="mt-2 text-base text-movo-slate">
          Sumá fotos de tu espacio y, si ya pagaste, el número de operación para que confirmemos tu
          alta más rápido.
        </p>
      </section>

      <form id="step-form" onSubmit={submit} noValidate className="space-y-5">
        <section className="space-y-4 rounded-3xl bg-white p-5 shadow-sm">
          <div className="flex items-center justify-between">
            <h3 className="flex items-center gap-2 font-display text-lg font-bold">
              <ImageIcon className="size-5 text-movo-gold" />
              Fotos del local
            </h3>
            <span className="rounded-full bg-movo-sand px-3 py-1 text-xs font-bold">
              {photos.length} de {MAX_PHOTOS} subidas
            </span>
          </div>

          {cover ? (
            <figure className="relative overflow-hidden rounded-3xl bg-movo-sand">
              {/* Las URLs firmadas cambian en cada carga: el optimizador de next/image no tiene qué cachear. */}
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={cover.url}
                alt="Foto de portada del local"
                className="h-56 w-full object-cover"
              />
              <span className="absolute top-3 left-3 inline-flex items-center gap-1 rounded-full bg-movo-accent px-3 py-1 text-xs font-bold">
                <StarIcon className="size-3.5" />
                PORTADA
              </span>
              <RemoveButton
                label="Quitar la portada"
                onClick={() => void gallery.remove(cover.id)}
                disabled={gallery.busy}
              />
            </figure>
          ) : null}

          {others.length > 0 ? (
            <ul className="grid grid-cols-2 gap-3">
              {others.map((photo, index) => (
                <li key={photo.id} className="relative overflow-hidden rounded-2xl bg-movo-sand">
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img
                    src={photo.url}
                    alt={`Foto ${index + 2} del local`}
                    className="h-32 w-full object-cover"
                  />
                  <RemoveButton
                    label={`Quitar la foto ${index + 2}`}
                    onClick={() => void gallery.remove(photo.id)}
                    disabled={gallery.busy}
                  />
                  <button
                    type="button"
                    onClick={() => void gallery.makeCover(photo.id)}
                    disabled={gallery.busy}
                    className="absolute bottom-2 left-2 rounded-full bg-movo-card/90 px-3 py-1 text-xs font-bold text-white outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-60"
                  >
                    Hacer portada
                  </button>
                </li>
              ))}
            </ul>
          ) : null}

          {photos.length < MAX_PHOTOS ? (
            <label
              className={`flex cursor-pointer flex-col items-center gap-2 rounded-3xl border-2 border-dashed border-movo-slate/40 px-4 py-8 text-center focus-within:ring-2 focus-within:ring-movo-accent ${
                gallery.busy ? "opacity-60" : ""
              }`}
            >
              <span className="grid size-12 place-items-center rounded-full bg-movo-sand">
                <CameraIcon />
              </span>
              <span className="font-semibold">
                {gallery.busy ? "Subiendo…" : "+ Agregar más fotos"}
              </span>
              <span className="text-sm text-movo-slate">JPG, PNG o WebP hasta 5 MB por foto</span>
              <input
                type="file"
                accept={PHOTO_TYPES.join(",")}
                multiple
                disabled={gallery.busy}
                className="sr-only"
                onChange={(event) => {
                  const files = Array.from(event.target.files ?? []);
                  event.target.value = "";
                  void gallery.upload(files);
                }}
              />
            </label>
          ) : null}

          {gallery.message ? (
            <Notice title={gallery.message.title} body={gallery.message.body} />
          ) : null}
        </section>

        <p className="rounded-3xl bg-movo-accent/20 p-5 text-sm leading-relaxed">
          <strong>Consejo:</strong> sumá al menos 3 fotos con buena luz. Es lo primero que ven los
          clientes cuando te encuentran.
        </p>

        <section className="space-y-4 rounded-3xl bg-white p-5 shadow-sm">
          <h3 className="font-display text-lg font-bold">Comprobante de pago</h3>
          <Field
            id="payment-reference"
            label="Número de operación"
            error={referenceError}
            hint={<span className="text-xs font-semibold text-movo-slate">Opcional</span>}
            help="Está en el comprobante de Mercado Pago. Nos ayuda a confirmar tu alta más rápido."
          >
            <TextInput
              id="payment-reference"
              value={reference}
              onChange={setReference}
              error={referenceError}
              disabled={pending}
              inputMode="numeric"
              maxLength={100}
              placeholder="98451238472"
            />
          </Field>
        </section>

        {message ? <Notice title={message.title} body={message.body} /> : null}
      </form>

      <StepFooter
        onBack={onBack}
        nextLabel="Enviar a revisión"
        pending={pending}
        disabled={gallery.busy}
      />
    </Shell>
  );
}

function RemoveButton({
  label,
  onClick,
  disabled,
}: {
  label: string;
  onClick: () => void;
  disabled: boolean;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      onClick={onClick}
      disabled={disabled}
      className="absolute top-2 right-2 grid size-9 place-items-center rounded-full bg-white/90 text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-60"
    >
      <XIcon className="size-4" />
    </button>
  );
}
