"use client";

import { useEffect, useState } from "react";

import type { ApiError } from "@/lib/session/types";

import { featuredNearby, searchNearby } from "./api";
import type { Point, SearchResult } from "./types";

/** El pilotaje arranca en Río Tercero (SDD §12): si no hay ubicación, se busca desde acá. */
export const DEFAULT_POINT: Point = { lat: -32.1738, lng: -64.1146 };

export type LocationState =
  { status: "locating" } | { status: "granted"; point: Point } | { status: "unavailable" };

/**
 * La ubicación del navegador, pedida una sola vez al montar. Si el permiso se niega o el
 * navegador no la da, `unavailable`: la pantalla usa `DEFAULT_POINT` y avisa. `locate`
 * vuelve a pedirla (por ejemplo desde el botón "Usar mi ubicación").
 */
export function useLocation(): { location: LocationState; locate: () => void } {
  const [location, setLocation] = useState<LocationState>({ status: "locating" });
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    if (!("geolocation" in navigator)) {
      const timer = setTimeout(() => setLocation({ status: "unavailable" }), 0);
      return () => clearTimeout(timer);
    }
    navigator.geolocation.getCurrentPosition(
      ({ coords }) =>
        setLocation({ status: "granted", point: { lat: coords.latitude, lng: coords.longitude } }),
      () => setLocation({ status: "unavailable" }),
      { enableHighAccuracy: false, timeout: 10_000, maximumAge: 60_000 },
    );
  }, [attempt]);

  function locate() {
    setLocation({ status: "locating" });
    setAttempt((n) => n + 1);
  }

  return { location, locate };
}

export type Results =
  | { status: "loading" }
  | { status: "ready"; items: SearchResult[] }
  | { status: "failed"; error: ApiError };

interface Loaded {
  key: string;
  results: Results;
}

function useResults(
  point: Point | null,
  extra: string,
  fetchItems: (
    point: Point,
  ) => Promise<{ ok: true; items: SearchResult[] } | { ok: false; error: ApiError }>,
): Results {
  const [loaded, setLoaded] = useState<Loaded | null>(null);
  const key = point ? `${point.lat},${point.lng},${extra}` : "";

  useEffect(() => {
    if (!point) return;
    let ignore = false;

    void fetchItems(point).then((result) => {
      if (ignore) return;
      setLoaded({
        key,
        results: result.ok
          ? { status: "ready", items: result.items }
          : { status: "failed", error: result.error },
      });
    });

    return () => {
      ignore = true;
    };
    // `key` resume el punto y el filtro: es lo único que dispara una búsqueda nueva.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);

  return loaded && loaded.key === key ? loaded.results : { status: "loading" };
}

/** Locales cerca de un punto, filtrados por categoría (slug). `point` nulo espera sin pedir nada. */
export function useNearby(point: Point | null, category: string | null): Results {
  return useResults(point, category ?? "", async (p) => {
    const result = await searchNearby(p, category);
    return result.ok
      ? { ok: true, items: result.data.results }
      : { ok: false, error: result.error };
  });
}

/** "Destacados cerca tuyo": lista vacía hasta que haya locales Pro (Fase 5). */
export function useFeatured(point: Point | null): Results {
  return useResults(point, "featured", async (p) => {
    const result = await featuredNearby(p);
    return result.ok
      ? { ok: true, items: result.data.results }
      : { ok: false, error: result.error };
  });
}
