"use client";

import { PASSWORD_MIN_LENGTH } from "@/lib/session/messages";
import { useAccessForm, useRedirectIfSignedIn } from "@/lib/session/hooks";

import { AuthField } from "./AuthField";
import { FormAlert } from "./FormAlert";
import { ArrowRightIcon, HeadsetIcon, LockIcon, MailIcon } from "./icons";

type LoginFormProps = {
  /** Enlace de soporte por WhatsApp; sin número configurado no se muestra. */
  supportUrl: string | null;
};

export function LoginForm({ supportUrl }: LoginFormProps) {
  useRedirectIfSignedIn();
  const form = useAccessForm("login");
  const { feedback } = form;

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
        label="Email comercial"
        hint={<span className="text-xs font-bold text-movo-accent">requerido</span>}
        icon={<MailIcon />}
        tone="dark"
        value={form.email}
        onChange={form.setEmail}
        autoComplete="username"
        error={feedback.fields.email}
        disabled={form.pending}
      />

      <AuthField
        id="password"
        label="Contraseña"
        hint={
          <span className="text-xs font-bold text-movo-muted">
            {PASSWORD_MIN_LENGTH}+ caracteres
          </span>
        }
        icon={<LockIcon />}
        tone="dark"
        type="password"
        value={form.password}
        onChange={form.setPassword}
        autoComplete="current-password"
        error={feedback.fields.password}
        disabled={form.pending}
      />

      {feedback.form ? <FormAlert title={feedback.form.title} body={feedback.form.body} /> : null}

      <button
        type="submit"
        disabled={form.pending}
        className="flex h-14 w-full items-center justify-center gap-2 rounded-full bg-movo-accent font-display text-lg font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-white disabled:opacity-60"
      >
        {form.pending ? "Ingresando…" : "Ingresar al panel"}
        {form.pending ? null : <ArrowRightIcon />}
      </button>

      {supportUrl ? (
        <div className="space-y-4 pt-1">
          <a
            href={supportUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="block text-center text-base font-semibold hover:text-movo-accent"
          >
            ¿Olvidaste tu contraseña?
          </a>
          <div className="mx-auto h-0.5 w-16 rounded-full bg-movo-muted/30" aria-hidden="true" />
          <a
            href={supportUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="mx-auto flex w-fit items-center gap-2 rounded-full bg-movo-tabs px-4 py-2.5 text-sm text-movo-muted hover:text-white"
          >
            <HeadsetIcon className="size-5 text-movo-accent" />
            Soporte exclusivo · WhatsApp de ayuda
          </a>
        </div>
      ) : null}
    </form>
  );
}
