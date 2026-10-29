"use client";

import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

import { Logo } from "@/components/Logo";
import { LogOutIcon, StarIcon, StoreIcon } from "@/components/socios/icons";
import { useSession } from "@/lib/session/SessionProvider";

const NAV = [
  { href: "/admin/locales", label: "Locales", Icon: StoreIcon },
  { href: "/admin/resenas", label: "Reseñas", Icon: StarIcon },
];

/**
 * Marco del panel del super admin: barra lateral oscura y contenido ancho. Hoy hay "Locales"
 * y "Reseñas"; Suscripciones y Métricas se suman con las fases 5 y 6.
 */
export function AdminShell({ children }: { children: ReactNode }) {
  const { state, logout } = useSession();
  const pathname = usePathname();
  const email = state.status === "authenticated" ? state.user.email : "";

  return (
    <div className="flex min-h-dvh flex-col bg-movo-cream font-sans text-movo-ink lg:flex-row">
      <aside className="flex shrink-0 flex-col bg-movo-page text-white lg:w-64">
        <div className="flex items-center gap-3 px-6 py-5">
          <Logo className="h-9" />
          <span className="rounded-full bg-movo-card px-3 py-1 text-xs font-bold tracking-wide text-movo-accent">
            SUPER ADMIN
          </span>
        </div>

        <nav aria-label="Secciones" className="px-4 pb-4">
          {NAV.map(({ href, label, Icon }) => {
            const current = pathname.startsWith(href);
            return (
              <a
                key={href}
                href={href}
                aria-current={current ? "page" : undefined}
                className={`flex items-center gap-3 rounded-xl px-4 py-3 font-semibold ${
                  current ? "bg-movo-accent text-movo-ink" : "text-white"
                }`}
              >
                <Icon className="size-5" />
                {label}
              </a>
            );
          })}
        </nav>

        <div className="mt-auto border-t border-white/10 p-4">
          <p className="truncate text-sm font-semibold">{email}</p>
          <button
            type="button"
            onClick={() => void logout()}
            className="mt-3 flex h-11 w-full items-center justify-center gap-2 rounded-full bg-movo-card text-sm font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
          >
            <LogOutIcon className="size-4" />
            Cerrar sesión
          </button>
        </div>
      </aside>

      <main className="min-w-0 flex-1 px-4 py-6 lg:px-8">{children}</main>
    </div>
  );
}
