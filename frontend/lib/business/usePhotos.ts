"use client";

import { useState } from "react";

import { presignPhoto, registerPhoto, removePhoto, reorderPhotos, uploadFile } from "./api";
import { describeFailure, type FormMessage } from "./messages";
import { coverFirst, moveInOrder } from "./photoOrder";
import { MAX_PHOTO_BYTES, PHOTO_TYPES, type Photo } from "./types";

/** Subida, orden, portada y borrado de las fotos del local (onboarding y panel). */
export function usePhotos(photos: Photo[], max: number, onChanged: () => void) {
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<FormMessage | null>(null);
  // Cuántos archivos de la tanda siguen sin subirse, para mostrar su lugar mientras tanto.
  const [pending, setPending] = useState(0);
  // Lo que no llegó a subirse por un error del servidor o de la red: es lo que "Reintentar" repite.
  const [failed, setFailed] = useState<File[]>([]);

  /** Sube los archivos de a uno: pide la URL firmada, sube directo al storage y los registra. */
  async function upload(files: File[]) {
    if (busy || files.length === 0) return;

    setBusy(true);
    setMessage(null);
    setFailed([]);
    setPending(files.length);

    let stored = photos.length;
    for (const [index, file] of files.entries()) {
      if (stored >= max) {
        setMessage({ title: "Ya subiste todas las fotos", body: "Borrá alguna para subir otra." });
        break;
      }
      const problem = check(file);
      if (problem) {
        setMessage(problem);
        setPending((n) => n - 1);
        continue;
      }

      const failure = await uploadOne(file);
      if (failure) {
        setMessage(failure);
        // Este archivo y los que faltaban quedan para reintentar.
        setFailed(files.slice(index).filter((rest) => check(rest) === null));
        break;
      }
      stored += 1;
      setPending((n) => n - 1);
    }

    setPending(0);
    setBusy(false);
    onChanged();
  }

  /** Repite la subida de lo que falló. */
  const retry = () => upload(failed);

  async function applyOrder(ids: number[] | null) {
    if (busy || ids === null) return;
    setBusy(true);
    setMessage(null);

    const result = await reorderPhotos(ids);
    if (!result.ok) setMessage(describeFailure(result).form ?? null);

    setBusy(false);
    onChanged();
  }

  const ids = photos.map((photo) => photo.id);

  /** La portada es la primera foto: elegirla es ponerla al frente del orden. */
  const makeCover = (id: number) => applyOrder(coverFirst(ids, id));

  /** Mueve la foto un lugar antes (`-1`) o después (`1`). */
  const move = (id: number, step: -1 | 1) => applyOrder(moveInOrder(ids, id, step));

  async function remove(id: number) {
    if (busy) return;
    setBusy(true);
    setMessage(null);

    const result = await removePhoto(id);
    if (!result.ok) setMessage(describeFailure(result).form ?? null);

    setBusy(false);
    onChanged();
  }

  return { busy, pending, failed, message, upload, retry, makeCover, move, remove };
}

/** Un mensaje si el archivo no sirve, o `null` si se puede subir. */
function check(file: File): FormMessage | null {
  if (!PHOTO_TYPES.includes(file.type)) {
    return { title: "Ese archivo no es una foto válida", body: "Usá una foto JPG, PNG o WebP." };
  }
  if (file.size === 0 || file.size > MAX_PHOTO_BYTES) {
    return { title: "La foto pesa demasiado", body: "Cada foto puede pesar hasta 5 MB." };
  }
  return null;
}

async function uploadOne(file: File): Promise<FormMessage | null> {
  const presigned = await presignPhoto(file.type, file.size);
  if (!presigned.ok) return describeFailure(presigned).form ?? null;

  const uploaded = await uploadFile(presigned.data.upload, file);
  if (!uploaded.ok) return describeFailure(uploaded).form ?? null;

  const registered = await registerPhoto(presigned.data.object_key);
  return registered.ok ? null : (describeFailure(registered).form ?? null);
}
