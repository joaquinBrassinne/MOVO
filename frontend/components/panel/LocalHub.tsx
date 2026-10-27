import Link from "next/link";

import { ArrowRightIcon, ClockIcon, ImageIcon, MapPinIcon } from "@/components/socios/icons";

const SECTIONS = [
  {
    href: "/socios/sedes",
    title: "Sedes y horarios",
    body: "Direcciones, horarios y qué sedes ve el público.",
    icon: MapPinIcon,
  },
  {
    href: "/socios/servicios",
    title: "Servicios",
    body: "Lo que ofrecés, sus precios y los días en que se dan.",
    icon: ClockIcon,
  },
  {
    href: "/socios/fotos",
    title: "Fotos",
    body: "La galería que se ve en la ficha de tu local.",
    icon: ImageIcon,
  },
] as const;

/** Mi local: la puerta a todo lo que el dueño configura de su negocio. */
export function LocalHub() {
  return (
    <div className="space-y-5">
      <div>
        <h2 className="font-display text-3xl leading-tight font-extrabold">Mi local</h2>
        <p className="mt-1 text-base text-movo-slate">
          Configurá lo que los clientes ven de tu negocio.
        </p>
      </div>

      <ul className="space-y-3">
        {SECTIONS.map(({ href, title, body, icon: Icon }) => (
          <li key={href}>
            <Link
              href={href}
              className="flex items-center gap-4 rounded-3xl bg-white p-5 shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
            >
              <span className="grid size-12 shrink-0 place-items-center rounded-full bg-movo-accent/40">
                <Icon className="size-6" />
              </span>
              <span className="min-w-0 flex-1">
                <span className="block font-display text-lg font-bold">{title}</span>
                <span className="block text-sm text-movo-slate">{body}</span>
              </span>
              <ArrowRightIcon className="size-5 shrink-0 text-movo-slate" />
            </Link>
          </li>
        ))}
      </ul>
    </div>
  );
}
