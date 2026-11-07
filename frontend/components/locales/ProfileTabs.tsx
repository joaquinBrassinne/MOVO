"use client";

import { useEffect, useState } from "react";

import { cn } from "@/lib/cn";

import { BUTTON_FOCUS, DIVIDER } from "./profileStyles";

export interface ProfileTab {
  /** Id de la sección a la que lleva (sin el #). */
  id: string;
  label: string;
}

/** Pestañas fijas de la ficha: enlaces a cada apartado, con el que se está leyendo marcado. */
export function ProfileTabs({ tabs }: { tabs: ProfileTab[] }) {
  const [active, setActive] = useState(tabs[0]?.id ?? "");

  useEffect(() => {
    const sections = tabs
      .map((tab) => document.getElementById(tab.id))
      .filter((el): el is HTMLElement => el !== null);
    // La franja de lectura está arriba: el apartado que la cruza es el activo.
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries.filter((entry) => entry.isIntersecting);
        if (visible.length > 0) setActive(visible[0].target.id);
      },
      { rootMargin: "-20% 0px -70% 0px" },
    );
    sections.forEach((section) => observer.observe(section));
    return () => observer.disconnect();
  }, [tabs]);

  return (
    <nav
      aria-label="Secciones de la ficha"
      className={cn(
        "no-scrollbar sticky top-0 z-20 mt-6 flex gap-6 overflow-x-auto border-b bg-movo-tabs px-5",
        DIVIDER,
      )}
    >
      {tabs.map((tab) => (
        <a
          key={tab.id}
          href={`#${tab.id}`}
          aria-current={active === tab.id ? "location" : undefined}
          className={cn(
            "shrink-0 border-b-2 py-3.5 text-[0.9375rem] font-semibold whitespace-nowrap",
            BUTTON_FOCUS,
            active === tab.id
              ? "border-movo-accent text-white"
              : "border-transparent text-movo-muted hover:text-white",
          )}
        >
          {tab.label}
        </a>
      ))}
    </nav>
  );
}
