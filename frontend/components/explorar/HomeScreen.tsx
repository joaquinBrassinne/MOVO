"use client";

import Link from "next/link";
import { useState } from "react";

import { Logo } from "@/components/Logo";
import { ArrowRightIcon, StarIcon } from "@/components/socios/icons";
import { useCategories } from "@/lib/business/hooks";
import { cn } from "@/lib/cn";
import { formatDistance } from "@/lib/search/format";
import { DEFAULT_POINT, useFeatured, useLocation, useNearby } from "@/lib/search/hooks";
import type { SearchResult } from "@/lib/search/types";

import { AccountLink, AppFrame } from "./AppFrame";
import { CategoryChips } from "./CategoryChips";
import { CategoryIcon, Icon, VerifiedIcon } from "./icons";

const profilePath = (result: SearchResult) => `/locales/${result.business_slug}`;

/** Home: buscador, categorías, "Destacados cerca tuyo" y "Cerca tuyo". */
export function HomeScreen() {
  const [category, setCategory] = useState<string | null>(null);
  const { location, locate } = useLocation();
  const categories = useCategories();

  const point =
    location.status === "granted"
      ? location.point
      : location.status === "unavailable"
        ? DEFAULT_POINT
        : null;
  const nearby = useNearby(point, category);
  const featured = useFeatured(point);
  const featuredItems = featured.status === "ready" ? featured.items : [];

  return (
    <AppFrame>
      <header className="flex items-center justify-between px-5 pt-5">
        <Logo className="h-9" />
        <AccountLink />
      </header>

      <div className="px-5 pt-5">
        <Link
          href="/mapa"
          className="flex h-14 items-center gap-3 rounded-full bg-movo-cream px-5 text-movo-slate outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
        >
          <Icon name="pin" className="size-5 text-movo-ink" />
          <span className="text-base">Mirá los locales en el mapa</span>
        </Link>
      </div>

      {location.status === "unavailable" ? (
        <div className="mx-5 mt-4 flex items-center justify-between gap-3 rounded-2xl bg-movo-card p-4 text-sm">
          <p className="text-movo-muted">
            No pudimos ubicarte, así que te mostramos locales de Río Tercero.
          </p>
          <button
            type="button"
            onClick={locate}
            className="shrink-0 rounded-full bg-movo-accent px-4 py-2 font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-white"
          >
            Usar mi ubicación
          </button>
        </div>
      ) : null}

      <div className="pt-5">
        {categories.status === "ready" ? (
          <CategoryChips
            variant="home"
            categories={categories.data}
            selected={category}
            onSelect={setCategory}
          />
        ) : null}
      </div>

      {featuredItems.length > 0 ? (
        <section aria-labelledby="featured-title" className="pt-6">
          <h2 id="featured-title" className="px-5 font-display text-xl font-bold">
            Destacados cerca tuyo
          </h2>
          <ul className="no-scrollbar mt-3 flex snap-x gap-4 overflow-x-auto px-5 pb-1">
            {featuredItems.map((item) => (
              <li key={item.branch_id} className="w-[18.5rem] shrink-0 snap-start">
                <FeaturedCard item={item} />
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      <section aria-labelledby="nearby-title" className="px-5 pt-8 pb-6">
        <div className="flex items-baseline justify-between">
          <h2 id="nearby-title" className="font-display text-xl font-bold">
            Cerca tuyo
          </h2>
          <Link
            href="/mapa"
            className="inline-flex items-center gap-1 text-sm font-bold text-movo-accent outline-none focus-visible:underline"
          >
            Ver en el mapa
            <ArrowRightIcon className="size-4" />
          </Link>
        </div>

        <NearbyList results={nearby} />
      </section>
    </AppFrame>
  );
}

function NearbyList({ results }: { results: ReturnType<typeof useNearby> }) {
  if (results.status === "loading") {
    return (
      <ul className="mt-4 space-y-4" aria-busy="true" aria-label="Cargando locales">
        {[0, 1, 2].map((n) => (
          <li key={n} className="h-28 animate-pulse rounded-3xl bg-movo-card" />
        ))}
      </ul>
    );
  }
  if (results.status === "failed") {
    return (
      <p role="alert" className="mt-4 rounded-2xl bg-movo-error-bg p-4 text-sm text-movo-error">
        No pudimos cargar los locales. Probá de nuevo en un rato.
      </p>
    );
  }
  if (results.items.length === 0) {
    return (
      <p className="mt-4 rounded-2xl bg-movo-card p-4 text-sm text-movo-muted">
        Todavía no hay locales cerca. Probá con otra categoría o mirá el mapa.
      </p>
    );
  }
  return (
    <ul className="mt-4 space-y-4">
      {results.items.map((item) => (
        <li key={item.branch_id}>
          <NearbyCard item={item} />
        </li>
      ))}
    </ul>
  );
}

function FeaturedCard({ item }: { item: SearchResult }) {
  return (
    <Link
      href={profilePath(item)}
      className="relative flex h-[26rem] flex-col justify-end overflow-hidden rounded-[2rem] bg-gradient-to-b from-movo-card to-movo-tabs p-5 outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
    >
      <span className="absolute top-5 left-5 inline-flex items-center gap-1.5 rounded-full bg-movo-tabs/90 px-3 py-1.5 text-xs font-bold tracking-wide uppercase">
        <StarIcon className="size-3.5" />
        Destacado
      </span>
      {item.verified ? (
        <span className="absolute top-5 right-5 grid size-10 place-items-center rounded-full bg-movo-accent text-movo-ink">
          <VerifiedIcon className="size-6" />
        </span>
      ) : null}
      <CategoryIcon
        slug={item.category_slug}
        className="absolute top-1/3 left-1/2 size-24 -translate-x-1/2 -translate-y-1/2 text-white/15"
      />
      <h3 className="font-display text-2xl leading-tight font-extrabold">{item.business_name}</h3>
      <p className="mt-1 flex items-center gap-2 text-sm text-movo-muted">
        <Icon name="pin" className="size-4" />
        {item.category_name} · a {formatDistance(item.distance_m)}
      </p>
    </Link>
  );
}

function NearbyCard({ item }: { item: SearchResult }) {
  return (
    <Link
      href={profilePath(item)}
      className="flex items-center gap-4 rounded-3xl bg-movo-card p-4 outline-none focus-visible:ring-2 focus-visible:ring-movo-accent"
    >
      <span
        className={cn(
          "grid size-[7.5rem] shrink-0 place-items-center rounded-2xl bg-gradient-to-br from-movo-tabs to-movo-page text-movo-accent",
        )}
      >
        <CategoryIcon slug={item.category_slug} className="size-12" />
      </span>
      <span className="min-w-0 flex-1">
        <span className="flex items-center gap-2">
          <span className="truncate font-display text-lg font-extrabold">{item.business_name}</span>
          {item.verified ? <VerifiedIcon className="size-5 shrink-0 text-movo-accent" /> : null}
        </span>
        <span className="mt-1 block truncate text-sm text-movo-muted">
          {item.category_name} · a {formatDistance(item.distance_m)}
        </span>
        <span className="mt-2 block truncate text-sm font-semibold text-movo-accent">
          {item.address}, {item.city}
        </span>
      </span>
      <ArrowRightIcon className="size-5 shrink-0" />
    </Link>
  );
}
