/** "450 m" bajo el kilómetro (redondeado a 50 m), "1.4 km" desde ahí con un decimal. */
export function formatDistance(meters: number): string {
  const rounded = Math.max(50, Math.round(meters / 50) * 50);
  if (rounded < 1000) return `${rounded} m`;
  return `${(meters / 1000).toFixed(1)} km`;
}
