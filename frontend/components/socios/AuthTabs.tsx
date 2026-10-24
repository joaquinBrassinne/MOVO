import Link from "next/link";

import { LogInIcon, UserPlusIcon } from "./icons";

type AuthTabsProps = {
  active: "login" | "register";
  /** La vista de ingreso lleva un ícono en cada pestaña; la de alta no. */
  withIcons?: boolean;
};

const TABS = [
  { id: "login", label: "Ingresar", href: "/socios/ingresar", Icon: LogInIcon },
  { id: "register", label: "Crear cuenta", href: "/socios/crear-cuenta", Icon: UserPlusIcon },
] as const;

export function AuthTabs({ active, withIcons = false }: AuthTabsProps) {
  return (
    <nav aria-label="Acceso socios" className="flex gap-1 rounded-full bg-movo-tabs p-1">
      {TABS.map(({ id, label, href, Icon }) => {
        const selected = id === active;

        return (
          <Link
            key={id}
            href={href}
            aria-current={selected ? "page" : undefined}
            className={`flex h-11 flex-1 items-center justify-center gap-2 rounded-full text-base font-semibold outline-none focus-visible:ring-2 focus-visible:ring-movo-accent ${
              selected ? "bg-movo-accent text-movo-ink" : "text-movo-muted hover:text-white"
            }`}
          >
            {withIcons ? <Icon /> : null}
            {label}
          </Link>
        );
      })}
    </nav>
  );
}
