"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import type { ReactNode } from "react";

import { Svg, UserIcon } from "@/components/socios/icons";
import { cn } from "@/lib/cn";

const TABS = [
  {
    href: "/",
    label: "Inicio",
    icon: '<circle cx="12" cy="12" r="10"/><path d="m16.2 7.8-2.100 6.300-6.300 2.100 2.100-6.300 6.300-2.100Z"/>',
  },
  {
    href: "/map",
    label: "Mapa",
    icon: '<path d="M14.1 4.6 9 3 3 5v16l6-2 5.100 1.600L21 19V3l-6.900 1.600Z"/><path d="M9 3v16"/><path d="M15 5v16"/>',
  },
  {
    href: "/coupons",
    label: "Mis cupones",
    icon: '<path d="M2 9a3 3 0 0 1 0 6v2a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2v-2a3 3 0 0 1 0-6V7a2 2 0 0 0-2-2H4a2 2 0 0 0-2 2Z"/><path d="M13 5v2"/><path d="M13 17v2"/><path d="M13 11v2"/>',
  },
] as const;

/** Botón redondo que lleva al ingreso de socios (los clientes no tienen cuenta). */
export function AccountLink({ className }: { className?: string }) {
  return (
    <Link
      href="/owners"
      aria-label="Ingreso de socios"
      className={cn(
        "grid size-11 place-items-center rounded-full bg-movo-card text-white outline-none focus-visible:ring-2 focus-visible:ring-movo-accent",
        className,
      )}
    >
      <UserIcon />
    </Link>
  );
}

/** Marco de Home y Mapa: contenido y barra inferior con las dos pantallas (Inicio y Mapa). */
export function AppFrame({ children }: { children: ReactNode }) {
  const pathname = usePathname();

  return (
    <div className="flex min-h-dvh flex-col bg-movo-page font-sans text-white">
      <div className="flex-1 pb-20">{children}</div>
      <nav
        aria-label="Principal"
        className="fixed inset-x-0 bottom-0 z-[1100] border-t border-white/10 bg-movo-card"
      >
        <ul className="mx-auto flex max-w-md">
          {TABS.map((tab) => {
            const active = pathname === tab.href;
            return (
              <li key={tab.href} className="flex-1">
                <Link
                  href={tab.href}
                  aria-current={active ? "page" : undefined}
                  className={cn(
                    "flex h-[4.5rem] flex-col items-center justify-center gap-1 text-xs font-bold outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-movo-accent",
                    active ? "text-movo-accent" : "text-movo-muted",
                  )}
                >
                  <Svg className="size-6">
                    <g dangerouslySetInnerHTML={{ __html: tab.icon }} />
                  </Svg>
                  {tab.label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>
    </div>
  );
}
