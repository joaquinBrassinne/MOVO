"use client";

import { useState, type ReactNode } from "react";

import { AlertCircleIcon, EyeIcon, EyeOffIcon } from "./icons";

type AuthFieldProps = {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: "email" | "password";
  autoComplete: string;
  /** Texto chico a la derecha de la etiqueta ("requerido", "10+ caracteres"). */
  hint?: ReactNode;
  /** Ícono a la izquierda del campo oscuro. */
  icon?: ReactNode;
  /** "dark" es el campo del ingreso; "light" el del alta. */
  tone: "dark" | "light";
  /** Etiqueta en mayúsculas chicas (alta) en lugar de título (ingreso). */
  caps?: boolean;
  error?: string;
  /** Mensaje que acompaña al campo cuando no hay error (la regla de la contraseña). */
  help?: ReactNode;
  disabled?: boolean;
};

export function AuthField({
  id,
  label,
  value,
  onChange,
  type = "email",
  autoComplete,
  hint,
  icon,
  tone,
  caps = false,
  error,
  help,
  disabled,
}: AuthFieldProps) {
  const [revealed, setRevealed] = useState(false);
  const isPassword = type === "password";
  const describedBy = error ? `${id}-error` : help ? `${id}-help` : undefined;

  const shell =
    tone === "dark"
      ? "rounded-xl bg-movo-field text-white"
      : error
        ? "rounded-full border-2 border-red-700 bg-movo-error-bg text-white"
        : "rounded-full bg-stone-100 text-movo-ink";

  return (
    <div>
      <div className="mb-2 flex items-baseline justify-between gap-3">
        <label
          htmlFor={id}
          className={caps ? "text-xs font-bold tracking-wide uppercase" : "text-base font-semibold"}
        >
          {label}
        </label>
        {hint}
      </div>

      <div
        className={`flex h-12 items-center gap-3 px-4 focus-within:ring-2 focus-within:ring-movo-accent ${shell}`}
      >
        {icon ? <span className="text-movo-muted">{icon}</span> : null}
        <input
          id={id}
          name={id}
          type={isPassword && revealed ? "text" : type}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          autoComplete={autoComplete}
          disabled={disabled}
          aria-invalid={error ? true : undefined}
          aria-describedby={describedBy}
          className="min-w-0 flex-1 bg-transparent text-base outline-none placeholder:text-movo-muted/60"
        />
        {isPassword ? (
          <button
            type="button"
            onClick={() => setRevealed((current) => !current)}
            aria-label={revealed ? "Ocultar contraseña" : "Mostrar contraseña"}
            aria-pressed={revealed}
            className="text-movo-muted outline-none hover:text-movo-accent focus-visible:text-movo-accent"
          >
            {revealed ? <EyeOffIcon /> : <EyeIcon />}
          </button>
        ) : error && tone === "light" ? (
          <AlertCircleIcon className="size-5 text-red-300" />
        ) : null}
      </div>

      {error ? (
        <p
          id={`${id}-error`}
          role="alert"
          className="mt-2 flex items-center gap-2 text-sm text-movo-error"
        >
          <AlertCircleIcon className="size-4 shrink-0" />
          {error}
        </p>
      ) : help ? (
        <div id={`${id}-help`} className="mt-2 text-sm text-movo-muted">
          {help}
        </div>
      ) : null}
    </div>
  );
}
