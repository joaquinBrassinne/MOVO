"use client";

import { useEffect, useRef, useState } from "react";

import { PhotoCoverBadge, UploadTile } from "@/components/business/PhotoParts";
import { Notice } from "@/components/onboarding/fields";
import { ChevronDownIcon, ChevronUpIcon, ImageIcon, MoreIcon } from "@/components/socios/icons";
import { useBusiness } from "@/lib/business/BusinessProvider";
import type { Photo } from "@/lib/business/types";
import { usePhotos } from "@/lib/business/usePhotos";

import { ConfirmDialog } from "./ConfirmDialog";

const ICON_BUTTON =
  "grid size-9 place-items-center rounded-full bg-white/90 text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-40";

/** Fotos del negocio: agregar, ordenar, elegir la portada y eliminar. Son del negocio, no de cada sede. */
export function PhotosPanel() {
  const { business, reload } = useBusiness();

  if (business.status === "loading") return <PhotosSkeleton />;
  if (business.status !== "ready") {
    return (
      <Notice
        title="No pudimos cargar tus fotos"
        body="Probá de nuevo en un rato. Si todavía no creaste tu local, empezá por el alta."
      />
    );
  }
  return (
    <Gallery photos={business.data.images} max={business.data.max_images} onChanged={reload} />
  );
}

function Gallery({
  photos,
  max,
  onChanged,
}: {
  photos: Photo[];
  max: number;
  onChanged: () => void;
}) {
  const gallery = usePhotos(photos, max, onChanged);
  const [removing, setRemoving] = useState<Photo | null>(null);
  const full = photos.length >= max;

  return (
    <div className="space-y-5">
      <section className="space-y-1">
        <h2 className="font-display text-3xl font-extrabold">Fotos del local</h2>
        <p className="text-movo-slate">
          Es lo primero que ven los clientes. La primera foto es la portada.
        </p>
      </section>

      <div className="flex items-center justify-between">
        <h3 className="flex items-center gap-2 font-display text-lg font-bold">
          <ImageIcon className="size-5 text-movo-gold" />
          Tus fotos
        </h3>
        <span className="rounded-full bg-movo-sand px-3 py-1 text-sm font-bold" aria-live="polite">
          {photos.length} de {max} fotos
        </span>
      </div>

      {photos.length === 0 && gallery.pending === 0 ? (
        <p className="rounded-3xl bg-white p-5 text-movo-slate shadow-sm">
          Todavía no subiste fotos. Una buena portada ayuda a que te elijan.
        </p>
      ) : null}

      <ul className="grid grid-cols-2 gap-3">
        {photos.map((photo, index) => (
          <li key={`${photo.id}-${index}`}>
            <PhotoCard
              photo={photo}
              index={index}
              last={index === photos.length - 1}
              busy={gallery.busy}
              onMove={(step) => void gallery.move(photo.id, step)}
              onCover={() => void gallery.makeCover(photo.id)}
              onRemove={() => setRemoving(photo)}
            />
          </li>
        ))}
        {Array.from({ length: gallery.pending }, (_, n) => (
          <li
            key={`uploading-${n}`}
            aria-label="Subiendo una foto"
            className="grid h-44 animate-pulse place-items-center rounded-3xl bg-movo-sand text-sm font-semibold text-movo-slate"
          >
            Subiendo…
          </li>
        ))}
        <li className="col-span-2">
          <UploadTile
            busy={gallery.busy}
            disabled={full}
            title={full ? "Llegaste al máximo de fotos" : "+ Agregar foto"}
            hint={full ? "Eliminá alguna para subir otra." : "JPG, PNG o WebP hasta 5 MB por foto"}
            onFiles={(files) => void gallery.upload(files)}
          />
        </li>
      </ul>

      {gallery.message ? (
        <div className="space-y-3">
          <Notice title={gallery.message.title} body={gallery.message.body} />
          {gallery.failed.length > 0 ? (
            <button
              type="button"
              onClick={() => void gallery.retry()}
              disabled={gallery.busy}
              className="h-12 rounded-full bg-movo-accent px-6 font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink disabled:opacity-60"
            >
              Reintentar
            </button>
          ) : null}
        </div>
      ) : null}

      <ConfirmDialog
        open={removing !== null}
        title="¿Eliminar esta foto?"
        body="Se quita de tu local y también se borra el archivo. No se puede deshacer."
        confirmLabel="Eliminar"
        onCancel={() => setRemoving(null)}
        onConfirm={() => {
          if (removing) void gallery.remove(removing.id);
          setRemoving(null);
        }}
      />
    </div>
  );
}

function PhotoCard({
  photo,
  index,
  last,
  busy,
  onMove,
  onCover,
  onRemove,
}: {
  photo: Photo;
  index: number;
  last: boolean;
  busy: boolean;
  onMove: (step: -1 | 1) => void;
  onCover: () => void;
  onRemove: () => void;
}) {
  const number = index + 1;
  return (
    <figure className="relative overflow-hidden rounded-3xl bg-movo-sand shadow-sm">
      {/* Las URLs firmadas cambian en cada carga: el optimizador de next/image no tiene qué cachear. */}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src={photo.url}
        alt={index === 0 ? "Foto de portada del local" : `Foto ${number} del local`}
        className="h-44 w-full object-cover"
      />
      {index === 0 ? <PhotoCoverBadge /> : null}

      <div className="absolute top-2 right-2">
        <PhotoMenu
          label={`Opciones de la foto ${number}`}
          disabled={busy}
          canCover={index > 0}
          onCover={onCover}
          onRemove={onRemove}
        />
      </div>

      <div className="absolute bottom-2 left-2 flex gap-1.5">
        <button
          type="button"
          aria-label={`Subir la foto ${number}`}
          onClick={() => onMove(-1)}
          disabled={busy || index === 0}
          className={ICON_BUTTON}
        >
          <ChevronUpIcon className="size-4" />
        </button>
        <button
          type="button"
          aria-label={`Bajar la foto ${number}`}
          onClick={() => onMove(1)}
          disabled={busy || last}
          className={ICON_BUTTON}
        >
          <ChevronDownIcon className="size-4" />
        </button>
      </div>
    </figure>
  );
}

/** El menú "⋯" de cada foto: elegirla como portada o eliminarla. Se cierra con Escape o al tocar afuera. */
function PhotoMenu({
  label,
  disabled,
  canCover,
  onCover,
  onRemove,
}: {
  label: string;
  disabled: boolean;
  canCover: boolean;
  onCover: () => void;
  onRemove: () => void;
}) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const close = (event: Event) => {
      if (event instanceof KeyboardEvent && event.key !== "Escape") return;
      if (event instanceof PointerEvent && root.current?.contains(event.target as Node)) return;
      setOpen(false);
    };
    document.addEventListener("keydown", close);
    document.addEventListener("pointerdown", close);
    return () => {
      document.removeEventListener("keydown", close);
      document.removeEventListener("pointerdown", close);
    };
  }, [open]);

  const item =
    "block w-full px-4 py-3 text-left text-sm font-semibold outline-none hover:bg-movo-sand focus-visible:bg-movo-sand";

  return (
    <div ref={root} className="relative">
      <button
        type="button"
        aria-label={label}
        aria-haspopup="menu"
        aria-expanded={open}
        disabled={disabled}
        onClick={() => setOpen((current) => !current)}
        className={ICON_BUTTON}
      >
        <MoreIcon className="size-4" />
      </button>
      {open ? (
        <div
          role="menu"
          className="absolute right-0 z-10 mt-1 w-48 overflow-hidden rounded-2xl bg-white shadow-lg"
        >
          {canCover ? (
            <button
              type="button"
              role="menuitem"
              className={item}
              onClick={() => {
                setOpen(false);
                onCover();
              }}
            >
              Usar como portada
            </button>
          ) : null}
          <button
            type="button"
            role="menuitem"
            className={`${item} text-movo-danger`}
            onClick={() => {
              setOpen(false);
              onRemove();
            }}
          >
            Eliminar
          </button>
        </div>
      ) : null}
    </div>
  );
}

function PhotosSkeleton() {
  return (
    <div aria-busy="true" aria-label="Cargando tus fotos" className="space-y-5">
      <div className="h-24 animate-pulse rounded-3xl bg-white" />
      <div className="grid grid-cols-2 gap-3">
        {[0, 1, 2, 3].map((n) => (
          <div key={n} className="h-44 animate-pulse rounded-3xl bg-movo-sand" />
        ))}
      </div>
    </div>
  );
}
