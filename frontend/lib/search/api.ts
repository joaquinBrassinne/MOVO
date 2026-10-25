import { requestPublic } from "@/lib/api/request";

import type { FeaturedList, Point, SearchPage } from "./types";

export const SEARCH_RADIUS_M = 5000;
export const SEARCH_LIMIT = 20;
export const FEATURED_LIMIT = 6;

function near(point: Point, extra: Record<string, string>): string {
  const params = new URLSearchParams({ lat: String(point.lat), lng: String(point.lng), ...extra });
  return params.toString();
}

/** Locales públicos cerca de un punto, con la categoría opcional (slug). */
export function searchNearby(point: Point, category: string | null) {
  const extra: Record<string, string> = {
    radius_m: String(SEARCH_RADIUS_M),
    limit: String(SEARCH_LIMIT),
  };
  if (category) extra.category = category;
  return requestPublic<SearchPage>(`/search?${near(point, extra)}`);
}

/** "Destacados cerca tuyo": vacío mientras no haya locales Pro (Fase 5). */
export function featuredNearby(point: Point) {
  return requestPublic<FeaturedList>(
    `/home/featured?${near(point, { radius_m: String(SEARCH_RADIUS_M), limit: String(FEATURED_LIMIT) })}`,
  );
}
