"use client";

import { useEffect, useRef } from "react";

interface Props {
  open: boolean;
  title: string;
  body: string;
  confirmLabel: string;
  onConfirm: () => void;
  onCancel: () => void;
}

/**
 * Confirmación modal sobre `<dialog>`: el navegador se ocupa del foco, de Escape y de
 * dejar el resto de la página inerte. Cerrarlo con Escape es cancelar.
 */
export function ConfirmDialog({ open, title, body, confirmLabel, onConfirm, onCancel }: Props) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  return (
    <dialog
      ref={ref}
      onCancel={(event) => {
        event.preventDefault();
        onCancel();
      }}
      aria-labelledby="confirm-title"
      className="m-auto w-[min(92vw,24rem)] rounded-3xl bg-white p-6 text-movo-ink shadow-xl backdrop:bg-movo-page/60"
    >
      <h2 id="confirm-title" className="font-display text-xl font-extrabold">
        {title}
      </h2>
      <p className="mt-2 text-sm text-movo-slate">{body}</p>
      <div className="mt-6 flex gap-3">
        <button
          type="button"
          onClick={onCancel}
          className="h-12 flex-1 rounded-full bg-movo-sand font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
        >
          Cancelar
        </button>
        <button
          type="button"
          onClick={onConfirm}
          className="h-12 flex-1 rounded-full bg-movo-danger font-bold text-white outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
        >
          {confirmLabel}
        </button>
      </div>
    </dialog>
  );
}
