"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState, type SubmitEvent } from "react";

import { describeAuthFailure, type AuthFeedback } from "./messages";
import { useSession } from "./SessionProvider";
import type { SessionUser } from "./types";

export const SIGN_IN_PATH = "/socios/ingresar";
export const PANEL_PATH = "/socios/panel";
export const ONBOARDING_PATH = "/socios/onboarding";
export const ADMIN_PATH = "/admin/locales";

/** La pantalla de inicio de cada rol: el super admin va a su directorio, el dueño a su panel. */
export function homePath(user: SessionUser): string {
  return user.role === "super_admin" ? ADMIN_PATH : PANEL_PATH;
}

/** Quien ya tiene sesión no necesita ver las vistas de acceso. */
export function useRedirectIfSignedIn() {
  const { state } = useSession();
  const router = useRouter();

  useEffect(() => {
    if (state.status === "authenticated") router.replace(homePath(state.user));
  }, [state, router]);
}

/** Las vistas privadas mandan a ingresar cuando no hay sesión. */
export function useRequireSession() {
  const { state } = useSession();
  const router = useRouter();

  useEffect(() => {
    if (state.status === "anonymous") router.replace(SIGN_IN_PATH);
  }, [state.status, router]);

  return state;
}

/**
 * Las vistas del super admin: sin sesión mandan a ingresar y a un dueño lo devuelven a su
 * panel. Devuelve `true` solo cuando la sesión es de un super admin. (La API vuelve a
 * exigir el rol en cada llamada; esto solo evita mostrar una pantalla que no le sirve.)
 */
export function useRequireSuperAdmin(): boolean {
  const state = useRequireSession();
  const router = useRouter();

  const isOwner = state.status === "authenticated" && state.user.role !== "super_admin";
  useEffect(() => {
    if (isOwner) router.replace(PANEL_PATH);
  }, [isOwner, router]);

  return state.status === "authenticated" && state.user.role === "super_admin";
}

/** Estado y envío de los formularios de ingreso y de alta. */
export function useAccessForm(mode: "login" | "register") {
  const { login, register } = useSession();
  const router = useRouter();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [pending, setPending] = useState(false);
  const [feedback, setFeedback] = useState<AuthFeedback>({ fields: {} });

  async function submit(event: SubmitEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending) return;

    if (mode === "register" && password !== confirm) {
      setFeedback({ fields: { confirm: "Las contraseñas no coinciden" } });
      return;
    }

    setPending(true);
    setFeedback({ fields: {} });

    const result = await (mode === "login" ? login : register)(email, password);
    if (result.ok) {
      // Se queda en "pending" hasta que la navegación termine: evita un doble envío.
      // Quien acaba de crear la cuenta empieza por el alta del negocio; quien ingresa va
      // a la pantalla de su rol.
      router.replace(mode === "register" ? ONBOARDING_PATH : homePath(result.data.user));
      return;
    }

    setFeedback(describeAuthFailure(result));
    setPending(false);
  }

  return { email, setEmail, password, setPassword, confirm, setConfirm, pending, feedback, submit };
}
