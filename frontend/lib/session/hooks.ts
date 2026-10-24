"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState, type SubmitEvent } from "react";

import { describeAuthFailure, type AuthFeedback } from "./messages";
import { useSession } from "./SessionProvider";

export const SIGN_IN_PATH = "/socios/ingresar";
export const PANEL_PATH = "/socios/panel";

/** Quien ya tiene sesión no necesita ver las vistas de acceso. */
export function useRedirectIfSignedIn() {
  const { state } = useSession();
  const router = useRouter();

  useEffect(() => {
    if (state.status === "authenticated") router.replace(PANEL_PATH);
  }, [state.status, router]);
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
      router.replace(PANEL_PATH);
      return;
    }

    setFeedback(describeAuthFailure(result));
    setPending(false);
  }

  return { email, setEmail, password, setPassword, confirm, setConfirm, pending, feedback, submit };
}
