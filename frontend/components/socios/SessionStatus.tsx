"use client";

import { useRouter } from "next/navigation";

import { SIGN_IN_PATH, useRequireSession } from "@/lib/session/hooks";
import { useSession } from "@/lib/session/SessionProvider";

const ROLE_LABEL = { business_admin: "Dueño de negocio", super_admin: "Super admin" } as const;

export function SessionStatus() {
  const state = useRequireSession();
  const { logout } = useSession();
  const router = useRouter();

  if (state.status !== "authenticated") {
    return <p className="text-movo-muted">Verificando tu sesión…</p>;
  }

  async function signOut() {
    await logout();
    router.replace(SIGN_IN_PATH);
  }

  return (
    <div className="space-y-4">
      <h1 className="font-display text-2xl font-extrabold">Sesión iniciada</h1>
      <dl className="space-y-1 text-movo-muted">
        <div className="flex gap-2">
          <dt className="font-semibold text-white">Email:</dt>
          <dd>{state.user.email}</dd>
        </div>
        <div className="flex gap-2">
          <dt className="font-semibold text-white">Rol:</dt>
          <dd>{ROLE_LABEL[state.user.role]}</dd>
        </div>
      </dl>
      <button
        type="button"
        onClick={signOut}
        className="h-12 w-full rounded-full bg-movo-accent font-display font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-white"
      >
        Cerrar sesión
      </button>
    </div>
  );
}
