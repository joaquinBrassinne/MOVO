"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState, type ReactNode } from "react";

import { QrIcon, TicketIcon } from "@/components/benefits/icons";
import { Logo } from "@/components/Logo";
import { GridIcon, LogOutIcon, StarIcon, StoreIcon, UserIcon } from "@/components/socios/icons";
import { useBusiness } from "@/lib/business/BusinessProvider";
import { useSession } from "@/lib/session/SessionProvider";

/** Las pestañas del panel; "Mi local" abarca la pantalla de accesos y las que cuelgan de ella. */
const NAV = [
  { href: "/socios/panel", label: "Inicio", icon: GridIcon, within: ["/socios/panel"] },
  { href: "/socios/canjear", label: "Canjear", icon: QrIcon, within: ["/socios/canjear"] },
  {
    href: "/socios/beneficios",
    label: "Beneficios",
    icon: TicketIcon,
    within: ["/socios/beneficios"],
  },
  { href: "/socios/reviews", label: "Reseñas", icon: StarIcon, within: ["/socios/reviews"] },
  {
    href: "/socios/local",
    label: "Mi local",
    icon: StoreIcon,
    within: ["/socios/local", "/socios/sedes", "/socios/servicios", "/socios/fotos"],
  },
] as const;

/** El título del encabezado según la vista en la que está el dueño. */
function titleOf(pathname: string): string {
  if (pathname === "/socios/beneficios") return "Beneficios";
  if (pathname === "/socios/canjear") return "Canjear cupón";
  if (pathname === "/socios/reviews") return "Reseñas";
  if (pathname === "/socios/local/datos") return "Datos del negocio";
  if (pathname === "/socios/local") return "Mi local";
  if (pathname === "/socios/servicios") return "Servicios y horarios";
  if (pathname === "/socios/fotos") return "Fotos del local";
  if (pathname === "/socios/sedes") return "Mis sedes";
  if (pathname === "/socios/sedes/nueva") return "Nueva sede";
  if (pathname.startsWith("/socios/sedes/")) return "Editar sede";
  return "Panel";
}

/** Marco del panel del dueño: encabezado oscuro con su negocio y navegación inferior. */
export function PanelChrome({ children }: { children: ReactNode }) {
  const pathname = usePathname();
  const { business } = useBusiness();
  const name = business.status === "ready" ? business.data.business.name : "";

  return (
    <div className="flex min-h-dvh flex-col bg-movo-cream font-sans text-movo-ink">
      <header className="sticky top-0 z-30 bg-movo-page text-white">
        <div className="mx-auto flex h-20 w-full max-w-md items-center justify-between gap-3 px-5">
          <Logo className="h-9" />
          <div className="min-w-0 flex-1 text-right">
            <h1 className="truncate font-display text-lg leading-tight font-bold">
              {titleOf(pathname)}
            </h1>
            <p className="truncate text-xs font-semibold tracking-wide text-movo-muted uppercase">
              Socios{name ? ` · ${name}` : ""}
            </p>
          </div>
          <AccountMenu />
        </div>
      </header>

      <main className="mx-auto w-full max-w-md flex-1 px-4 pt-5 pb-28">{children}</main>

      <nav
        aria-label="Secciones del panel"
        className="fixed inset-x-0 bottom-0 z-30 border-t border-movo-sand bg-white"
      >
        <ul className="mx-auto flex max-w-md justify-around px-2 py-2">
          {NAV.map(({ href, label, icon: Icon, within }) => {
            const current = within.some(
              (prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`),
            );
            return (
              <li key={href} className="min-w-0 flex-1">
                <Link
                  href={href}
                  aria-current={current ? "page" : undefined}
                  className={`flex w-full flex-col items-center gap-1 rounded-2xl py-2 text-[11px] font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent ${
                    current ? "bg-movo-accent/30 text-movo-ink" : "text-movo-slate"
                  }`}
                >
                  <Icon className="size-6" />
                  {label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>
    </div>
  );
}

/** El ícono de la cuenta abre quién está ingresado y la salida. */
function AccountMenu() {
  const { state, logout } = useSession();
  const [open, setOpen] = useState(false);
  const email = state.status === "authenticated" ? state.user.email : "";

  return (
    <div className="relative">
      <button
        type="button"
        aria-label="Cuenta"
        aria-expanded={open}
        onClick={() => setOpen((current) => !current)}
        className="grid size-10 place-items-center rounded-full bg-movo-card outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
      >
        <UserIcon />
      </button>

      {open ? (
        <div className="absolute top-12 right-0 w-64 rounded-2xl bg-white p-4 text-movo-ink shadow-lg">
          <p className="truncate text-sm font-semibold">{email}</p>
          <button
            type="button"
            onClick={() => void logout()}
            className="mt-3 flex h-11 w-full items-center justify-center gap-2 rounded-full bg-movo-sand font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
          >
            <LogOutIcon className="size-4" />
            Cerrar sesión
          </button>
        </div>
      ) : null}
    </div>
  );
}
