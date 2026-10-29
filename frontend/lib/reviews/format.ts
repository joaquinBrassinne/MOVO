import { MIN_REVIEWS_TO_RATE } from "./types.ts";

/** Con menos de 3 reseñas el local se muestra como "Nuevo", sin estrellas (SDD 5.5). */
export const isNewBranch = (count: number): boolean => count < MIN_REVIEWS_TO_RATE;

/** "4,6": un decimal y coma decimal. */
export function formatRating(average: number): string {
  return average.toFixed(1).replace(".", ",");
}

/** Cuánto de cada una de las cinco estrellas está llena: 0, 0.5 o 1 (medias estrellas). */
export function starFills(average: number): number[] {
  const value = Math.min(5, Math.max(0, average));
  return Array.from({ length: 5 }, (_, index) => {
    const part = Math.min(1, Math.max(0, value - index));
    return Math.round(part * 2) / 2;
  });
}
