"use client";

import Link from "next/link";

import { PASSWORD_MIN_LENGTH } from "@/lib/session/messages";
import { useAccessForm, useRedirectIfSignedIn } from "@/lib/session/hooks";

import { AuthField } from "./AuthField";
import { FormAlert } from "./FormAlert";
import { ArrowRightIcon, CheckCircleIcon } from "./icons";

export function RegisterForm() {
  useRedirectIfSignedIn();
  const form = useAccessForm("register");
  const { feedback } = form;
  const longEnough = form.password.length >= PASSWORD_MIN_LENGTH;

  return (
    <form
      method="post"
      onSubmit={form.submit}
      noValidate
      aria-busy={form.pending}
      className="space-y-5"
    >
      <AuthField
        id="email"
        label="Email"
        caps
        tone="light"
        value={form.email}
        onChange={form.setEmail}
        autoComplete="username"
        error={feedback.fields.email}
        disabled={form.pending}
      />

      <AuthField
        id="password"
        label="Contraseña"
        caps
        tone="light"
        type="password"
        value={form.password}
        onChange={form.setPassword}
        autoComplete="new-password"
        error={feedback.fields.password}
        help={
          <span className="flex items-center gap-2">
            <CheckCircleIcon
              className={`size-4 ${longEnough ? "text-movo-accent" : "text-movo-muted/60"}`}
            />
            Mínimo {PASSWORD_MIN_LENGTH} caracteres
          </span>
        }
        disabled={form.pending}
      />

      <AuthField
        id="confirm"
        label="Repetir contraseña"
        caps
        tone="light"
        type="password"
        value={form.confirm}
        onChange={form.setConfirm}
        autoComplete="new-password"
        error={feedback.fields.confirm}
        disabled={form.pending}
      />

      {feedback.form ? <FormAlert title={feedback.form.title} body={feedback.form.body} /> : null}

      <button
        type="submit"
        disabled={form.pending}
        className="flex h-14 w-full items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold text-movo-ink shadow-[0_0_24px_rgb(252_211_77/0.25)] outline-none focus-visible:ring-2 focus-visible:ring-white disabled:opacity-60"
      >
        {form.pending ? "Creando cuenta…" : "Crear cuenta"}
        {form.pending ? null : <ArrowRightIcon />}
      </button>

      <p className="pt-2 text-center text-base text-movo-muted">
        ¿Ya tenés cuenta?{" "}
        <Link
          href="/socios/ingresar"
          className="font-semibold text-movo-accent underline underline-offset-4"
        >
          Ingresar
        </Link>
      </p>
    </form>
  );
}
