"use client";

import { useRef, useState } from "react";

import { ArrowLeftIcon, ArrowRightIcon } from "@/components/socios/icons";

interface Props {
  images: string[];
  name: string;
}

const FOCUS = "outline-none focus-visible:ring-2 focus-visible:ring-movo-ink";

/**
 * Apartado "Fotos" de la ficha: una grilla de miniaturas cuadradas, como la del alta del local, y al tocar una se abre entera en un visor (con flechas y Esc para cerrar).
 * Sin fotos no se dibuja nada.
 */
export function PhotoSection({ images, name }: Props) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [current, setCurrent] = useState(0);

  if (images.length === 0) return null;

  const open = (index: number) => {
    setCurrent(index);
    dialog.current?.showModal();
  };
  const step = (delta: number) => setCurrent((i) => (i + delta + images.length) % images.length);
  const label = (index: number) => `${name}, foto ${index + 1} de ${images.length}`;

  return (
    <section aria-labelledby="photos" className="mt-8">
      <div className="flex items-baseline justify-between">
        <h2 id="photos" className="font-display text-xl font-bold">
          Fotos
        </h2>
        <span className="text-sm text-movo-slate">
          {images.length} {images.length === 1 ? "foto" : "fotos"}
        </span>
      </div>

      <ul className="mt-3 grid grid-cols-3 gap-2 sm:grid-cols-[repeat(auto-fill,7rem)] sm:gap-3">
        {images.map((src, index) => (
          <li key={src}>
            <button
              type="button"
              onClick={() => open(index)}
              aria-label={`Ver ${label(index)} en grande`}
              className={`block aspect-square w-full cursor-pointer overflow-hidden rounded-2xl bg-movo-sand shadow-sm ${FOCUS}`}
            >
              {/* Las URLs son prefirmadas y de otro origen: next/image no las optimiza. */}
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img
                src={src}
                alt=""
                loading="lazy"
                className="size-full object-cover transition-transform duration-300 motion-safe:hover:scale-105"
              />
            </button>
          </li>
        ))}
      </ul>

      <dialog
        ref={dialog}
        aria-label={`Fotos de ${name}`}
        onClick={(event) => {
          if (event.target === dialog.current) dialog.current?.close();
        }}
        className="m-auto size-full max-h-dvh max-w-none bg-black/95 p-0 backdrop:bg-black"
      >
        <div className="relative grid size-full place-items-center p-4">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={images[current]}
            alt={label(current)}
            className="max-h-[75dvh] max-w-full rounded-2xl object-contain"
          />
          <button
            type="button"
            onClick={() => dialog.current?.close()}
            className="absolute top-4 right-4 min-h-11 cursor-pointer rounded-full bg-movo-cream px-5 text-sm font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
          >
            Cerrar
          </button>
          {images.length > 1 ? (
            <div className="absolute inset-x-0 bottom-6 flex items-center justify-center gap-5">
              <button
                type="button"
                aria-label="Foto anterior"
                onClick={() => step(-1)}
                className="grid size-12 cursor-pointer place-items-center rounded-full bg-movo-cream text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
              >
                <ArrowLeftIcon />
              </button>
              <p aria-live="polite" className="min-w-14 text-center text-sm font-bold text-white">
                {current + 1} / {images.length}
              </p>
              <button
                type="button"
                aria-label="Foto siguiente"
                onClick={() => step(1)}
                className="grid size-12 cursor-pointer place-items-center rounded-full bg-movo-cream text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
              >
                <ArrowRightIcon />
              </button>
            </div>
          ) : null}
        </div>
      </dialog>
    </section>
  );
}
