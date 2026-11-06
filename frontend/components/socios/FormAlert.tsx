import { AlertCircleIcon } from "./icons";

/** Aviso del formulario completo: credenciales incorrectas, límite de intentos, caídas. */
export function FormAlert({ title, body }: { title: string; body: string }) {
  return (
    <div role="alert" className="flex gap-3 rounded-lg bg-movo-error-bg p-4">
      <AlertCircleIcon className="mt-0.5 size-6 shrink-0 text-red-400" />
      <div>
        <p className="font-semibold text-red-200">{title}</p>
        <p className="mt-1 text-sm text-movo-muted">{body}</p>
      </div>
    </div>
  );
}
