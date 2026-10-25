"use client";

import { CameraIcon, XIcon } from "@/components/socios/icons";
import { PHOTO_TYPES } from "@/lib/business/types";

/** La "x" que quita una foto, pegada a su esquina superior derecha. */
export function RemovePhotoButton({
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

interface UploadTileProps {
  /** Lo que se está subiendo: atenúa el tile y cambia su texto. */
  busy: boolean;
  /** No se puede subir (por ejemplo, se llegó al máximo). */
  disabled?: boolean;
  title: string;
  hint: string;
  onFiles: (files: File[]) => void;
  /** Para que el tile ocupe una celda de la grilla en vez de todo el ancho. */
  compact?: boolean;
}

/** El tile punteado para elegir fotos del dispositivo. Valida el tipo en el selector del navegador. */
export function UploadTile({
  busy,
  disabled = false,
  title,
  hint,
  onFiles,
  compact,
}: UploadTileProps) {
  const inactive = busy || disabled;
  return (
    <label
      className={`flex flex-col items-center justify-center gap-2 rounded-3xl border-2 border-dashed border-movo-slate/40 text-center focus-within:ring-2 focus-within:ring-movo-accent ${
        compact ? "min-h-40 px-3 py-4" : "px-4 py-8"
      } ${inactive ? "cursor-not-allowed opacity-60" : "cursor-pointer"}`}
    >
      <span className="grid size-12 place-items-center rounded-full bg-movo-sand">
        <CameraIcon />
      </span>
      <span className="font-semibold">{title}</span>
      <span className="text-sm text-movo-slate">{hint}</span>
      <input
        type="file"
        accept={PHOTO_TYPES.join(",")}
        multiple
        disabled={inactive}
        className="sr-only"
        onChange={(event) => {
          const files = Array.from(event.target.files ?? []);
          event.target.value = "";
          onFiles(files);
        }}
      />
    </label>
  );
}
