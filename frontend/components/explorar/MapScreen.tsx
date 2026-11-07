"use client";

import dynamic from "next/dynamic";
import Link from "next/link";
import { useState } from "react";

import { CoverTile } from "@/components/CoverImage";
import { StarRating } from "@/components/reviews/StarRating";
import { ArrowRightIcon, StarIcon } from "@/components/socios/icons";
import { useCategories } from "@/lib/business/hooks";
import { cn } from "@/lib/cn";
import { distanceBetween } from "@/lib/search/geo";
import { formatDistance } from "@/lib/search/format";
import { DEFAULT_POINT, useLocation, useNearby } from "@/lib/search/hooks";
import type { Point, SearchResult } from "@/lib/search/types";

import { AccountLink, AppFrame } from "./AppFrame";
import { BenefitPill } from "./BenefitPill";
import { CategoryChips } from "./CategoryChips";
import { Icon, VerifiedIcon } from "./icons";

// Leaflet necesita `window`: el mapa se carga solo en el navegador.
const SearchMap = dynamic(() => import("./SearchMap"), {
  ssr: false,
  loading: () => <div className="size-full animate-pulse bg-movo-card" />,
});

/** Cuánto se tiene que mover el mapa para ofrecer "Buscar en esta zona". */
const SEARCH_AGAIN_M = 300;

const FOCUS = "outline-none focus-visible:ring-2 focus-visible:ring-movo-accent";

/** Home: mapa a pantalla completa con los locales cercanos y la lista en una hoja inferior. */
export function MapScreen() {
  const [category, setCategory] = useState<string | null>(null);
  const [searched, setSearched] = useState<Point | null>(null);
  const [viewed, setViewed] = useState<Point | null>(null);
  const [selected, setSelected] = useState<number | null>(null);
  const [expanded, setExpanded] = useState(false);
  const [recenter, setRecenter] = useState(0);
  const { location, locate } = useLocation();
  const categories = useCategories();

  const user = location.status === "granted" ? location.point : null;
  const located = user ?? (location.status === "unavailable" ? DEFAULT_POINT : null);
  // Mientras no se pidió "Buscar en esta zona", se busca desde donde está la persona.
  const point = searched ?? located;
  const results = useNearby(point, category);
  const items = results.status === "ready" ? results.items : [];

  const moved = point && viewed ? distanceBetween(point, viewed) > SEARCH_AGAIN_M : false;
  const area = items[0]?.city ?? null;

  function useMyLocation() {
    setSearched(null);
    setRecenter((n) => n + 1);
    locate();
  }

  return (
    <AppFrame>
      <div className="fixed inset-x-0 top-0 bottom-[4.5rem]">
        {point ? (
          <SearchMap
            center={point}
            recenterKey={`${point.lat},${point.lng},${recenter}`}
            results={items}
            selectedId={selected}
            user={user}
            onSelect={(id) => {
              setSelected(id);
              setExpanded(true);
            }}
            onMoved={setViewed}
          />
        ) : (
          <div
            className="size-full animate-pulse bg-movo-card"
            aria-label="Buscando tu ubicación"
          />
        )}
      </div>

      <div className="pointer-events-none fixed inset-x-0 top-0 z-[1000] space-y-3 px-4 pt-4">
        <div className="pointer-events-auto flex h-[3.25rem] items-center gap-2.5 rounded-2xl bg-white py-0 pr-2 pl-4 text-movo-ink shadow-[0_12px_30px_-10px_rgb(0_0_0/0.5)]">
          <Icon name="pin" className="size-5 shrink-0" />
          <p className="min-w-0 flex-1 truncate font-display text-base font-bold">
            {area ?? "Cerca tuyo"}
          </p>
          <button
            type="button"
            aria-label="Usar mi ubicación"
            onClick={useMyLocation}
            className={cn(
              "grid size-9 shrink-0 cursor-pointer place-items-center rounded-xl bg-movo-page text-white",
              FOCUS,
            )}
          >
            <Icon name="route" className="size-[1.125rem]" />
          </button>
          <AccountLink className="size-9 rounded-xl bg-movo-page" />
        </div>

        {categories.status === "ready" ? (
          <div className="pointer-events-auto -mx-4">
            <CategoryChips
              categories={categories.data}
              selected={category}
              onSelect={(slug) => {
                setCategory(slug);
                setSelected(null);
              }}
            />
          </div>
        ) : null}

        {moved ? (
          <div className="pointer-events-auto flex justify-center">
            <button
              type="button"
              onClick={() => {
                setSearched(viewed);
                setSelected(null);
              }}
              className="inline-flex h-11 cursor-pointer items-center gap-2 rounded-xl bg-movo-accent px-5 text-sm font-bold text-movo-ink shadow-lg outline-none focus-visible:ring-2 focus-visible:ring-white"
            >
              Buscar en esta zona
            </button>
          </div>
        ) : null}
      </div>

      <section
        aria-label="Locales cerca tuyo"
        className={cn(
          "fixed inset-x-0 bottom-[4.5rem] z-[1050] flex flex-col rounded-t-[1.75rem] bg-movo-tabs text-white shadow-[0_-12px_30px_rgb(0_0_0/0.35)] motion-safe:transition-[height] motion-safe:duration-300",
          expanded ? "h-[70dvh]" : "h-[17rem]",
        )}
      >
        <button
          type="button"
          aria-expanded={expanded}
          aria-label={expanded ? "Bajar la lista" : "Subir la lista"}
          onClick={() => setExpanded((open) => !open)}
          className={cn("grid h-8 shrink-0 cursor-pointer place-items-center", FOCUS)}
        >
          <span className="h-1.5 w-10 rounded-full bg-movo-muted/35" />
        </button>

        <div className="flex items-baseline justify-between gap-3 px-5">
          <h1 className="font-display text-[1.375rem] font-extrabold tracking-tight">Cerca tuyo</h1>
          {results.status === "ready" ? (
            <p className="text-sm text-movo-muted">
              {items.length} {items.length === 1 ? "local" : "locales"}
              {area ? ` · ${area}` : ""}
            </p>
          ) : null}
        </div>

        <div className="mt-3 flex-1 overflow-y-auto pb-4">
          <ResultList results={results} selected={selected} />
          <Link
            href="/plans"
            className={cn(
              "mx-5 mt-5 flex items-center justify-between gap-3 rounded-2xl p-4 ring-1 ring-movo-muted/20",
              FOCUS,
            )}
          >
            <span className="text-sm text-movo-muted">
              <span className="block font-bold text-white">¿Tenés un local?</span>
              Sumate a MOVO y aparecé en el mapa.
            </span>
            <span className="inline-flex shrink-0 items-center gap-1 text-sm font-bold text-movo-accent">
              Ver planes
              <ArrowRightIcon className="size-4" />
            </span>
          </Link>
        </div>
      </section>
    </AppFrame>
  );
}

function ResultList({
  results,
  selected,
}: {
  results: ReturnType<typeof useNearby>;
  selected: number | null;
}) {
  if (results.status === "loading") {
    return <p className="px-5 py-4 text-movo-muted">Buscando locales…</p>;
  }
  if (results.status === "failed") {
    return (
      <p role="alert" className="mx-5 rounded-2xl bg-movo-error-bg p-4 text-sm text-movo-error">
        No pudimos cargar los locales. Probá de nuevo en un rato.
      </p>
    );
  }
  if (results.items.length === 0) {
    return (
      <p className="mx-5 rounded-2xl bg-movo-card p-4 text-sm text-movo-muted">
        No hay locales en esta zona. Mové el mapa y buscá de nuevo, o probá con otra categoría.
      </p>
    );
  }
  // La tarjeta grande es el local elegido en el mapa; sin elección, el más cercano.
  const lead = results.items.find((item) => item.branch_id === selected) ?? results.items[0];
  const rest = results.items.filter((item) => item.branch_id !== lead.branch_id);
  return (
    <>
      <LeadCard item={lead} />
      <ul className="mt-1 px-4">
        {rest.map((item) => (
          <li key={item.branch_id}>
            <ResultRow item={item} />
          </li>
        ))}
      </ul>
    </>
  );
}

const profilePath = (item: SearchResult) => `/businesses/${item.business_slug}`;

/** Sello "Destacado" del plan Pro (visibilidad destacada). */
function FeaturedSeal({ className }: { className?: string }) {
  return (
    <span
      className={cn(
        "inline-flex w-fit items-center gap-1 rounded-md bg-movo-accent/15 px-1.5 py-0.5 text-xs font-bold text-movo-accent",
        className,
      )}
    >
      <StarIcon className="size-3" aria-hidden="true" />
      Destacado
    </span>
  );
}

function LeadCard({ item }: { item: SearchResult }) {
  return (
    <Link
      href={profilePath(item)}
      className={cn(
        "mx-4 flex items-center gap-3 rounded-[1.25rem] bg-movo-page p-3 ring-[1.5px] ring-movo-accent",
        FOCUS,
      )}
    >
      <CoverTile
        src={item.cover_url}
        categorySlug={item.category_slug}
        className="size-[5.25rem] shrink-0 rounded-[0.875rem] bg-movo-field"
        iconClassName="size-9"
      />
      <span className="min-w-0 flex-1">
        <span className="block truncate text-[0.8125rem] text-movo-muted">
          {item.category_name} · a {formatDistance(item.distance_m)}
        </span>
        {item.featured ? <FeaturedSeal className="mb-1" /> : null}
        <span className="flex items-center gap-1.5">
          <span className="truncate font-display text-[1.0625rem] font-extrabold tracking-tight">
            {item.business_name}
          </span>
          {item.verified ? <VerifiedIcon className="size-4 shrink-0 text-movo-accent" /> : null}
        </span>
        <StarRating
          average={item.rating_avg}
          count={item.rating_count}
          showCount
          onDark
          className="mt-0.5"
        />
        <BenefitPill label={item.benefit_label} className="mt-1.5" />
      </span>
      <span className="grid size-10 shrink-0 place-items-center rounded-xl bg-movo-accent text-movo-ink">
        <ArrowRightIcon className="size-[1.125rem]" />
      </span>
    </Link>
  );
}

function ResultRow({ item }: { item: SearchResult }) {
  return (
    <div className="flex items-center gap-3 border-b border-movo-muted/10 px-1 py-3">
      <CoverTile
        src={item.cover_url}
        categorySlug={item.category_slug}
        className="size-[3.25rem] shrink-0 rounded-xl bg-movo-field"
        iconClassName="size-6"
      />
      <Link href={profilePath(item)} className={cn("min-w-0 flex-1 rounded", FOCUS)}>
        <span className="block truncate font-display text-base font-extrabold tracking-tight">
          {item.business_name}
        </span>
        <span className="mt-0.5 flex items-center gap-1.5 text-[0.8125rem] text-movo-muted">
          {item.featured ? <FeaturedSeal /> : null}
          <span className="truncate">{item.category_name}</span>
          <StarRating
            average={item.rating_avg}
            count={item.rating_count}
            onDark
            className="shrink-0"
          />
        </span>
        <BenefitPill label={item.benefit_label} className="mt-1" />
      </Link>
      <span className="shrink-0 text-sm font-bold">a {formatDistance(item.distance_m)}</span>
    </div>
  );
}
