"use client";

import type { Category } from "@/lib/business/types";
import { cn } from "@/lib/cn";

import { Icon, CategoryIcon } from "./icons";

interface Props {
  categories: Category[];
  /** Slug elegido; `null` es "todas". */
  selected: string | null;
  onSelect: (slug: string | null) => void;
  /** Home: círculos con el nombre abajo. Mapa: píldoras con "Todos" primero. */
  variant: "home" | "map";
}

/** El filtro por categoría. Las categorías vienen de la API, así que una nueva aparece sola. */
export function CategoryChips({ categories, selected, onSelect, variant }: Props) {
  if (variant === "map") {
    return (
      <ul className="no-scrollbar flex gap-2 overflow-x-auto px-4 pb-1" aria-label="Categorías">
        <li>
          <button
            type="button"
            aria-pressed={selected === null}
            onClick={() => onSelect(null)}
            className={cn(
              "inline-flex h-11 shrink-0 items-center gap-2 rounded-full px-4 text-sm font-bold shadow-md outline-none focus-visible:ring-2 focus-visible:ring-movo-ink",
              selected === null ? "bg-movo-accent text-movo-ink" : "bg-white text-movo-ink",
            )}
          >
            <Icon name="bolt" className="size-5" />
            Todos
          </button>
        </li>
        {categories.map((category) => (
          <li key={category.slug}>
            <button
              type="button"
              aria-pressed={selected === category.slug}
              onClick={() => onSelect(selected === category.slug ? null : category.slug)}
              className={cn(
                "inline-flex h-11 shrink-0 items-center gap-2 rounded-full px-4 text-sm font-bold shadow-md outline-none focus-visible:ring-2 focus-visible:ring-movo-ink",
                selected === category.slug
                  ? "bg-movo-accent text-movo-ink"
                  : "bg-white text-movo-ink",
              )}
            >
              <CategoryIcon slug={category.slug} className="size-5" />
              {category.name}
            </button>
          </li>
        ))}
      </ul>
    );
  }

  return (
    <ul className="no-scrollbar flex gap-4 overflow-x-auto px-5 pb-1" aria-label="Categorías">
      {categories.map((category) => {
        const active = selected === category.slug;
        return (
          <li key={category.slug} className="shrink-0">
            <button
              type="button"
              aria-pressed={active}
              onClick={() => onSelect(active ? null : category.slug)}
              className="group flex w-20 flex-col items-center gap-2 text-xs font-bold outline-none"
            >
              <span
                className={cn(
                  "grid size-[4.5rem] place-items-center rounded-full ring-offset-2 ring-offset-movo-page group-focus-visible:ring-2 group-focus-visible:ring-movo-accent",
                  active ? "bg-movo-accent text-movo-ink" : "bg-movo-card text-white",
                )}
              >
                <CategoryIcon slug={category.slug} className="size-8" />
              </span>
              <span className={cn("text-center leading-tight", active && "text-movo-accent")}>
                {category.name}
              </span>
            </button>
          </li>
        );
      })}
    </ul>
  );
}
