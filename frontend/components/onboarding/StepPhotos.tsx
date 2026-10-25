"use client";

import { useState, type SubmitEvent } from "react";

import { ImageIcon, StarIcon } from "@/components/socios/icons";
import { submitBusiness } from "@/lib/business/api";
import { describeFailure, FIELD_MESSAGES, type FormMessage } from "@/lib/business/messages";
import { recalledReturnParams } from "@/lib/business/returnParams";
import type { Business, Photo } from "@/lib/business/types";
import { usePhotos } from "@/lib/business/usePhotos";

import { RemovePhotoButton, UploadTile } from "@/components/business/PhotoParts";

import { Field, Notice, TextInput } from "./fields";
import { Shell, StepFooter } from "./Shell";

interface Props {
  business: Business;
  photos: Photo[];
  /** Cuántas fotos puede tener el negocio. */
  maxPhotos: number;
  /** Se llama después de cambiar las fotos o de enviar el negocio, para volver a pedir los datos. */
  onChanged: () => void;
  onBack: () => void;
}

/** Paso 4: fotos del local y datos del pago en Mercado Pago; al terminar el negocio pasa a revisión. */
export function StepPhotos({ business, photos, maxPhotos, onChanged, onBack }: Props) {
  const gallery = usePhotos(photos, maxPhotos, onChanged);
  const [email, setEmail] = useState(business.mp_payer_email);
  const [name, setName] = useState(business.mp_payer_name);
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState<FormMessage | null>(null);
  const [emailError, setEmailError] = useState<string | undefined>();
  const [nameError, setNameError] = useState<string | undefined>();

  const [cover, ...others] = photos;

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending || gallery.busy) return;

    // La API informa un solo error por vez: así los dos campos vacíos se marcan juntos.
    if (!email.trim() || !name.trim()) {
      setEmailError(email.trim() ? undefined : FIELD_MESSAGES.mp_payer_email);
      setNameError(name.trim() ? undefined : FIELD_MESSAGES.mp_payer_name);
      return;
    }

    setPending(true);
    setMessage(null);
    setEmailError(undefined);
    setNameError(undefined);

    const result = await submitBusiness({ email, name, returnParams: recalledReturnParams() });
    setPending(false);

    if (result.ok) {
      onChanged();
      return;
    }
    const feedback = describeFailure(result);
    setEmailError(feedback.fields.mp_payer_email);
    setNameError(feedback.fields.mp_payer_name);
    setMessage(feedback.form ?? null);
  }

  return (
    <Shell step={4}>
      <section>
        <h2 className="font-display text-4xl font-extrabold">Fotos y pago</h2>
        <p className="mt-2 text-base text-movo-slate">
          Sumá fotos de tu espacio y completá con qué cuenta de Mercado Pago pagaste para que
          confirmemos tu alta más rápido.
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
              {photos.length} de {maxPhotos} subidas
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
              <RemovePhotoButton
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
                  <RemovePhotoButton
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

          {photos.length < maxPhotos ? (
            <UploadTile
              busy={gallery.busy}
              title={gallery.busy ? "Subiendo…" : "+ Agregar más fotos"}
              hint="JPG, PNG o WebP hasta 5 MB por foto"
              onFiles={(files) => void gallery.upload(files)}
            />
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
          <h3 className="font-display text-lg font-bold">Tu pago en Mercado Pago</h3>
          <p className="text-sm text-movo-slate">Los usamos para confirmar tu pago más rápido.</p>
          <Field
            id="mp-payer-email"
            label="Email con el que pagaste en Mercado Pago"
            required
            error={emailError}
          >
            <TextInput
              id="mp-payer-email"
              type="email"
              inputMode="email"
              autoComplete="email"
              value={email}
              onChange={setEmail}
              error={emailError}
              disabled={pending}
              maxLength={254}
              placeholder="tu@email.com"
            />
          </Field>
          <Field
            id="mp-payer-name"
            label="Nombre completo del titular (como figura en Mercado Pago)"
            required
            error={nameError}
          >
            <TextInput
              id="mp-payer-name"
              autoComplete="name"
              value={name}
              onChange={setName}
              error={nameError}
              disabled={pending}
              maxLength={100}
              placeholder="Ana Pérez"
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
