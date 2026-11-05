"use client";

import Link from "next/link";
import { useState } from "react";

import { Logo } from "@/components/Logo";
import { StarRating } from "@/components/reviews/StarRating";
import { ArrowRightIcon, StarIcon } from "@/components/socios/icons";
import { useCategories } from "@/lib/business/hooks";
import { formatDistance } from "@/lib/search/format";
import { DEFAULT_POINT, useFeatured, useLocation, useNearby } from "@/lib/search/hooks";
import type { SearchResult } from "@/lib/search/types";

import { AccountLink, AppFrame } from "./AppFrame";
import { BenefitPill } from "./BenefitPill";
import { CategoryChips } from "./CategoryChips";
import { CategoryIcon, Icon, VerifiedIcon } from "./icons";

const profilePath = (result: SearchResult) => `/businesses/${result.business_slug}`;

const STEPS = [
  { title: "Encontrá", text: "Mirá qué hay cerca tuyo, en la lista o en el mapa." },
  {
    title: "Generá tu cupón",
    text: "Dejá tu nombre y tu WhatsApp. No hace falta crear una cuenta.",
  },
  { title: "Canjealo en el local", text: "Mostrá el código del cupón y aprovechá tu beneficio." },
] as const;

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
      <div className="relative overflow-hidden">
        <div
          aria-hidden="true"
          className="pointer-events-none absolute -top-24 -right-16 size-72 rounded-full bg-movo-accent/20 blur-3xl"
        />
        <div
          aria-hidden="true"
          className="pointer-events-none absolute top-40 -left-24 size-64 rounded-full bg-movo-card blur-3xl"
        />

        <header className="relative flex items-center justify-between px-5 pt-5">
          <Logo className="h-9" />
          <AccountLink />
        </header>

        <div className="relative px-5 pt-8">
          <p className="text-sm font-bold tracking-widest text-movo-accent uppercase">
            Actividades y beneficios
          </p>
          <h1 className="mt-2 font-display text-[2rem] leading-[1.1] font-extrabold text-balance">
            Tu próxima actividad, <span className="text-movo-accent">cerca tuyo.</span>
          </h1>
          <p className="mt-3 max-w-sm text-base text-movo-muted">
            Gimnasios, pilates, yoga y más. Generá tu cupón y canjealo en el local.
          </p>

          <Link
            href="/map"
            className="mt-6 flex h-14 cursor-pointer items-center gap-3 rounded-full bg-movo-cream pr-2 pl-5 text-movo-slate shadow-lg shadow-black/20 outline-none transition-transform focus-visible:ring-2 focus-visible:ring-movo-accent motion-safe:active:scale-[0.99]"
          >
            <Icon name="pin" className="size-5 shrink-0 text-movo-ink" />
            <span className="flex-1 truncate text-base">Mirá los locales en el mapa</span>
            <span className="grid size-10 shrink-0 place-items-center rounded-full bg-movo-accent text-movo-ink">
              <ArrowRightIcon className="size-5" />
            </span>
          </Link>
        </div>

        {location.status === "unavailable" ? (
          <div className="relative mx-5 mt-4 flex items-center justify-between gap-3 rounded-2xl bg-movo-card p-4 text-sm">
            <p className="text-movo-muted">
              No pudimos ubicarte, así que te mostramos locales de Río Tercero.
            </p>
            <button
              type="button"
              onClick={locate}
              className="min-h-11 shrink-0 cursor-pointer rounded-full bg-movo-accent px-4 py-2 font-bold text-movo-ink outline-none focus-visible:ring-2 focus-visible:ring-white"
            >
              Usar mi ubicación
            </button>
          </div>
        ) : null}

        <div className="relative pt-7">
          {categories.status === "ready" ? (
            <CategoryChips
              variant="home"
              categories={categories.data}
              selected={category}
              onSelect={setCategory}
            />
          ) : null}
        </div>
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
            href="/map"
            className="inline-flex items-center gap-1 text-sm font-bold text-movo-accent outline-none focus-visible:underline"
          >
            Ver en el mapa
            <ArrowRightIcon className="size-4" />
          </Link>
        </div>

        <NearbyList results={nearby} />
      </section>

      <section aria-labelledby="how-title" className="px-5 pb-6">
        <h2 id="how-title" className="font-display text-xl font-bold">
          Cómo funciona
        </h2>
        <ol className="mt-4 grid gap-3">
          {STEPS.map((step, index) => (
            <li key={step.title} className="flex items-start gap-4 rounded-3xl bg-movo-card p-4">
              <span className="grid size-10 shrink-0 place-items-center rounded-full bg-movo-accent font-display text-base font-extrabold text-movo-ink">
                {index + 1}
              </span>
              <span>
                <span className="block font-display text-base font-extrabold">{step.title}</span>
                <span className="mt-0.5 block text-sm text-movo-muted">{step.text}</span>
              </span>
            </li>
          ))}
        </ol>
      </section>

      <section className="px-5 pb-8">
        <div className="rounded-[2rem] bg-gradient-to-br from-movo-accent to-amber-200 p-6 text-movo-ink">
          <h2 className="font-display text-xl leading-tight font-extrabold">¿Tenés un local?</h2>
          <p className="mt-1 text-sm font-medium">
            Sumate a MOVO, aparecé en el mapa y ofrecé beneficios a quienes están cerca.
          </p>
          <Link
            href="/plans"
            className="mt-4 inline-flex min-h-11 cursor-pointer items-center gap-2 rounded-full bg-movo-ink px-5 text-sm font-bold text-white outline-none focus-visible:ring-2 focus-visible:ring-movo-ink focus-visible:ring-offset-2 focus-visible:ring-offset-amber-200"
          >
            Ver planes
            <ArrowRightIcon className="size-4" />
          </Link>
        </div>
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
      className="relative flex h-[26rem] flex-col justify-end overflow-hidden rounded-[2rem] bg-gradient-to-b from-movo-card to-movo-tabs p-5 outline-none cursor-pointer ring-1 ring-white/5 focus-visible:ring-2 focus-visible:ring-movo-accent"
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
      {item.cover_url ? (
        <>
          {/* Las URLs son prefirmadas y de otro origen: next/image no las optimiza. */}
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={item.cover_url}
            alt=""
            loading="lazy"
            className="absolute inset-0 size-full object-cover"
          />
          <span
            aria-hidden="true"
            className="absolute inset-0 bg-gradient-to-t from-movo-tabs via-movo-tabs/60 to-transparent"
          />
        </>
      ) : (
        <CategoryIcon
          slug={item.category_slug}
          className="absolute top-1/3 left-1/2 size-24 -translate-x-1/2 -translate-y-1/2 text-white/15"
        />
      )}
      <BenefitPill label={item.benefit_label} className="mb-2 self-start" />
      <h3 className="font-display text-2xl leading-tight font-extrabold">{item.business_name}</h3>
      <StarRating
        average={item.rating_avg}
        count={item.rating_count}
        showCount
        onDark
        className="mt-1 self-start"
      />
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
      className="flex items-center gap-4 rounded-3xl bg-movo-card p-4 outline-none transition-colors hover:bg-movo-card/80 focus-visible:ring-2 focus-visible:ring-movo-accent cursor-pointer"
    >
      <span className="relative grid size-[7.5rem] shrink-0 place-items-center overflow-hidden rounded-2xl bg-gradient-to-br from-movo-tabs to-movo-page text-movo-accent">
        {item.cover_url ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={item.cover_url}
            alt=""
            loading="lazy"
            className="absolute inset-0 size-full object-cover"
          />
        ) : (
          <CategoryIcon slug={item.category_slug} className="size-12" />
        )}
      </span>
      <span className="min-w-0 flex-1">
        <span className="flex items-center gap-2">
          <span className="truncate font-display text-lg font-extrabold">{item.business_name}</span>
          {item.verified ? <VerifiedIcon className="size-5 shrink-0 text-movo-accent" /> : null}
        </span>
        <span className="mt-1 block truncate text-sm text-movo-muted">
          {item.category_name} · a {formatDistance(item.distance_m)}
        </span>
        <StarRating
          average={item.rating_avg}
          count={item.rating_count}
          showCount
          onDark
          className="mt-1"
        />
        <BenefitPill label={item.benefit_label} className="mt-2" />
        <span className="mt-2 block truncate text-sm font-semibold text-movo-accent">
          {item.address}, {item.city}
        </span>
      </span>
      <ArrowRightIcon className="size-5 shrink-0" />
    </Link>
  );
}
