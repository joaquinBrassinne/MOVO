"use client";

import dynamic from "next/dynamic";
import Link from "next/link";
import { useState } from "react";

import { Logo } from "@/components/Logo";
import { StarRating } from "@/components/reviews/StarRating";
import { ArrowLeftIcon, ArrowRightIcon, StarIcon } from "@/components/socios/icons";
import { useCategories } from "@/lib/business/hooks";
import { cn } from "@/lib/cn";
import { distanceBetween } from "@/lib/search/geo";
import { formatDistance } from "@/lib/search/format";
import { DEFAULT_POINT, useLocation, useNearby } from "@/lib/search/hooks";
import type { Point, SearchResult } from "@/lib/search/types";

import { AccountLink, AppFrame } from "./AppFrame";
import { BenefitPill } from "./BenefitPill";
import { CategoryChips } from "./CategoryChips";
import { CategoryIcon, VerifiedIcon } from "./icons";

// Leaflet necesita `window`: el mapa se carga solo en el navegador.
const SearchMap = dynamic(() => import("./SearchMap"), {
  ssr: false,
  loading: () => <div className="size-full animate-pulse bg-movo-card" />,
});

/** Cuánto se tiene que mover el mapa para ofrecer "Buscar en esta zona". */
const SEARCH_AGAIN_M = 300;

/** Mapa: locales cerca del punto elegido, filtro por categoría y lista en una hoja inferior. */
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
            onLocate={useMyLocation}
          />
        ) : (
          <div
            className="size-full animate-pulse bg-movo-card"
            aria-label="Buscando tu ubicación"
          />
        )}
      </div>

      <div className="pointer-events-none fixed inset-x-0 top-0 z-[1000] space-y-3 bg-gradient-to-b from-movo-page/90 to-transparent pt-4 pb-6">
        <header className="pointer-events-auto flex items-center justify-between px-5">
          <Logo className="h-8" />
          <AccountLink />
        </header>

        <div className="pointer-events-auto px-4">
          <div className="flex h-14 items-center gap-3 rounded-full bg-movo-cream px-3 text-movo-ink shadow-md">
            <Link
              href="/"
              aria-label="Volver al inicio"
              className="grid size-10 place-items-center rounded-full outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
            >
              <ArrowLeftIcon />
            </Link>
            <p className="truncate font-display text-lg font-bold">{area ?? "Cerca tuyo"}</p>
          </div>
        </div>

        <div className="pointer-events-auto">
          {categories.status === "ready" ? (
            <CategoryChips
              variant="map"
              categories={categories.data}
              selected={category}
              onSelect={(slug) => {
                setCategory(slug);
                setSelected(null);
              }}
            />
          ) : null}
        </div>

        {moved ? (
          <div className="pointer-events-auto flex justify-center">
            <button
              type="button"
              onClick={() => {
                setSearched(viewed);
                setSelected(null);
              }}
              className="inline-flex h-12 items-center gap-2 rounded-full bg-movo-accent px-6 font-bold text-movo-ink shadow-lg outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
            >
              Buscar en esta zona
            </button>
          </div>
        ) : null}
      </div>

      <section
        aria-label="Locales cerca tuyo"
        className={cn(
          "fixed inset-x-0 bottom-[4.5rem] z-[1050] flex flex-col rounded-t-[2rem] bg-movo-cream text-movo-ink shadow-[0_-8px_24px_rgb(0_0_0/0.25)] transition-[height] duration-300",
          expanded ? "h-[65dvh]" : "h-40",
        )}
      >
        <button
          type="button"
          aria-expanded={expanded}
          aria-label={expanded ? "Bajar la lista" : "Subir la lista"}
          onClick={() => setExpanded((open) => !open)}
          className="grid h-8 shrink-0 place-items-center outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
        >
          <span className="h-1.5 w-12 rounded-full bg-movo-slate/40" />
        </button>

        <div className="flex items-start justify-between gap-3 px-6">
          <div>
            <h2 className="font-display text-3xl font-extrabold">Cerca tuyo</h2>
            {area ? <p className="text-movo-slate">en {area}</p> : null}
          </div>
          {results.status === "ready" ? (
            <span className="rounded-full bg-movo-card px-4 py-1.5 text-sm font-bold text-movo-accent">
              {items.length} {items.length === 1 ? "disponible" : "disponibles"}
            </span>
          ) : null}
        </div>

        <div className="mt-3 flex-1 overflow-y-auto px-5 pb-4">
          <ResultList results={results} selected={selected} onSelect={setSelected} />
        </div>
      </section>
    </AppFrame>
  );
}

function ResultList({
  results,
  selected,
  onSelect,
}: {
  results: ReturnType<typeof useNearby>;
  selected: number | null;
  onSelect: (id: number) => void;
}) {
  if (results.status === "loading") {
    return <p className="py-4 text-movo-slate">Buscando locales…</p>;
  }
  if (results.status === "failed") {
    return (
      <p role="alert" className="rounded-2xl bg-white p-4 text-sm text-movo-danger">
        No pudimos cargar los locales. Probá de nuevo en un rato.
      </p>
    );
  }
  if (results.items.length === 0) {
    return (
      <p className="rounded-2xl bg-white p-4 text-sm text-movo-slate">
        No hay locales en esta zona. Mové el mapa y buscá de nuevo, o probá con otra categoría.
      </p>
    );
  }
  return (
    <ul className="space-y-3">
      {results.items.map((item) => (
        <li key={item.branch_id}>
          <ResultCard item={item} selected={item.branch_id === selected} onSelect={onSelect} />
        </li>
      ))}
    </ul>
  );
}

function ResultCard({
  item,
  selected,
  onSelect,
}: {
  item: SearchResult;
  selected: boolean;
  onSelect: (id: number) => void;
}) {
  return (
    <article
      onClick={() => onSelect(item.branch_id)}
      className={cn(
        "flex items-center gap-4 rounded-3xl bg-white p-4 shadow-sm",
        selected && "ring-2 ring-movo-accent",
      )}
    >
      <span className="grid size-20 shrink-0 place-items-center rounded-2xl bg-movo-card text-movo-accent">
        <CategoryIcon slug={item.category_slug} className="size-9" />
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap gap-2">
          {item.featured ? (
            <span className="inline-flex items-center gap-1 rounded-full bg-movo-accent/30 px-2.5 py-1 text-xs font-bold text-movo-gold">
              <StarIcon className="size-3.5" />
              Destacado
            </span>
          ) : null}
          {item.verified ? (
            <span className="inline-flex items-center gap-1 rounded-full px-1 py-1 text-xs font-bold text-movo-gold">
              <VerifiedIcon className="size-5" />
              Verificado
            </span>
          ) : null}
        </div>
        <h3 className="truncate font-display text-xl font-extrabold">{item.business_name}</h3>
        <p className="truncate text-sm text-movo-slate">{item.category_name}</p>
        <StarRating
          average={item.rating_avg}
          count={item.rating_count}
          showCount
          className="mt-1"
        />
        <BenefitPill label={item.benefit_label} className="mt-1" />
        <div className="mt-1 flex items-center justify-between gap-2">
          <span className="text-sm text-movo-slate">a {formatDistance(item.distance_m)}</span>
          <Link
            href={`/businesses/${item.business_slug}`}
            className="inline-flex items-center gap-1 font-bold outline-none focus-visible:underline"
          >
            Ver local
            <ArrowRightIcon className="size-5" />
          </Link>
        </div>
      </div>
    </article>
  );
}
