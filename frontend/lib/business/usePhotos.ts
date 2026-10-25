"use client";

import { useState } from "react";

import { presignPhoto, registerPhoto, removePhoto, reorderPhotos, uploadFile } from "./api";
import { describeFailure, type FormMessage } from "./messages";
import { MAX_PHOTO_BYTES, PHOTO_TYPES, type Photo } from "./types";

/** Subida, portada y borrado de las fotos del local (paso 4). */
export function usePhotos(photos: Photo[], max: number, onChanged: () => void) {
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<FormMessage | null>(null);

  /** Sube los archivos de a uno: pide la URL firmada, sube directo al storage y los registra. */
  async function upload(files: File[]) {
    if (busy || files.length === 0) return;

    setBusy(true);
    setMessage(null);

    let stored = photos.length;
    for (const file of files) {
      if (stored >= max) {
        setMessage({ title: "Ya subiste todas las fotos", body: "Borrá alguna para subir otra." });
        break;
      }
      const problem = check(file);
      if (problem) {
        setMessage(problem);
        continue;
      }

      const failure = await uploadOne(file);
      if (failure) {
        setMessage(failure);
        break;
      }
      stored += 1;
    }

    setBusy(false);
    onChanged();
  }

  /** La portada es la primera foto: elegirla es ponerla al frente del orden. */
  async function makeCover(id: number) {
    if (busy) return;
    setBusy(true);
    setMessage(null);

    const ids = [id, ...photos.map((p) => p.id).filter((other) => other !== id)];
    const result = await reorderPhotos(ids);
    if (!result.ok) setMessage(describeFailure(result).form ?? null);

    setBusy(false);
    onChanged();
  }

  async function remove(id: number) {
    if (busy) return;
    setBusy(true);
    setMessage(null);

    const result = await removePhoto(id);
    if (!result.ok) setMessage(describeFailure(result).form ?? null);

    setBusy(false);
    onChanged();
  }

  return { busy, message, upload, makeCover, remove };
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
