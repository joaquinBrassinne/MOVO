"use client";

import type { Category } from "@/lib/business/types";
import { cn } from "@/lib/cn";

import { CategoryIcon, Icon } from "./icons";

interface Props {
  categories: Category[];
  /** Slug elegido; `null` es "todas". */
  selected: string | null;
  onSelect: (slug: string | null) => void;
}

const CHIP =
  "inline-flex h-9 shrink-0 cursor-pointer items-center gap-1.5 rounded-xl px-3.5 text-sm font-semibold shadow-[0_6px_16px_-8px_rgb(0_0_0/0.6)] outline-none focus-visible:ring-2 focus-visible:ring-movo-accent";

/** El filtro por categoría. Las categorías vienen de la API, así que una nueva aparece sola. */
export function CategoryChips({ categories, selected, onSelect }: Props) {
  return (
    <ul className="no-scrollbar flex gap-2 overflow-x-auto px-4 pb-1" aria-label="Categorías">
      <li>
        <button
          type="button"
          aria-pressed={selected === null}
          onClick={() => onSelect(null)}
          className={cn(
            CHIP,
            selected === null ? "bg-movo-accent text-movo-ink" : "bg-movo-page text-white",
          )}
        >
          <Icon name="bolt" className="size-4" />
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
              CHIP,
              selected === category.slug
                ? "bg-movo-accent text-movo-ink"
                : "bg-movo-page text-white",
            )}
          >
            <CategoryIcon slug={category.slug} className="size-4" />
            {category.name}
          </button>
        </li>
      ))}
    </ul>
  );
}
