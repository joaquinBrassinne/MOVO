import type { ReactNode } from "react";

import { AlertCircleIcon } from "@/components/socios/icons";

interface FieldProps {
  id: string;
  label: string;
  /** Texto chico a la derecha de la etiqueta ("Opcional", "Obligatorio"). */
  hint?: ReactNode;
  required?: boolean;
  error?: string;
  /** Aclaración debajo del campo cuando no hay error. */
  help?: ReactNode;
  children: ReactNode;
}

/** Etiqueta, control y mensaje de un campo del panel claro. */
export function Field({ id, label, hint, required, error, help, children }: FieldProps) {
  return (
    <div>
      <div className="mb-2 flex items-baseline justify-between gap-3">
        <label htmlFor={id} className="text-base font-semibold text-movo-ink">
          {label}
          {required ? <span className="text-movo-danger"> *</span> : null}
        </label>
        {hint}
      </div>
      {children}
      {error ? (
        <p
          id={`${id}-error`}
          role="alert"
          className="mt-2 flex items-center gap-2 text-sm text-movo-danger"
        >
          <AlertCircleIcon className="size-4 shrink-0" />
          {error}
        </p>
      ) : help ? (
        <p id={`${id}-help`} className="mt-2 text-sm text-movo-slate">
          {help}
        </p>
      ) : null}
    </div>
  );
}

const CONTROL =
  "w-full rounded-2xl bg-movo-sand px-4 text-base text-movo-ink outline-none placeholder:text-movo-slate/60 focus:ring-2 focus:ring-movo-accent disabled:opacity-60";

interface InputProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
  disabled?: boolean;
  placeholder?: string;
  type?: "text" | "url" | "tel" | "time";
  autoComplete?: string;
  inputMode?: "text" | "tel" | "numeric" | "url";
  maxLength?: number;
}

export function TextInput({
  id,
  value,
  onChange,
  error,
  disabled,
  placeholder,
  type = "text",
  autoComplete,
  inputMode,
  maxLength,
}: InputProps) {
  return (
    <input
      id={id}
      name={id}
      type={type}
      value={value}
      onChange={(event) => onChange(event.target.value)}
      disabled={disabled}
      placeholder={placeholder}
      autoComplete={autoComplete}
      inputMode={inputMode}
      maxLength={maxLength}
      aria-invalid={error ? true : undefined}
      aria-describedby={error ? `${id}-error` : undefined}
      className={`h-14 ${CONTROL} ${error ? "ring-2 ring-movo-danger" : ""}`}
    />
  );
}

interface TextAreaProps {
  id: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
  disabled?: boolean;
  maxLength: number;
  rows?: number;
}

export function TextArea({
  id,
  value,
  onChange,
  error,
  disabled,
  maxLength,
  rows = 4,
}: TextAreaProps) {
  return (
    <textarea
      id={id}
      name={id}
      value={value}
      onChange={(event) => onChange(event.target.value)}
      disabled={disabled}
      maxLength={maxLength}
      rows={rows}
      aria-invalid={error ? true : undefined}
      aria-describedby={error ? `${id}-error` : undefined}
      className={`resize-none py-3 leading-relaxed ${CONTROL} ${error ? "ring-2 ring-movo-danger" : ""}`}
    />
  );
}

/** Aviso del formulario completo, en el panel claro. */
export function Notice({ title, body }: { title: string; body: string }) {
  return (
    <div role="alert" className="flex gap-3 rounded-2xl bg-movo-danger/10 p-4 text-movo-ink">
      <AlertCircleIcon className="mt-0.5 size-5 shrink-0 text-movo-danger" />
      <div>
        <p className="font-semibold">{title}</p>
        <p className="mt-1 text-sm text-movo-slate">{body}</p>
      </div>
    </div>
  );
}
