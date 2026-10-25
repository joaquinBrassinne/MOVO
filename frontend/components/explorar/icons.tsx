// Íconos de trazo fino de Home y Mapa. Cada uno es un trozo de SVG de 24×24 que sirve
// igual al componente de React y al marcador de Leaflet (que necesita HTML, no JSX):
// una sola fuente para los dos.

import { Svg } from "@/components/socios/icons";

const PATHS = {
  bolt: '<path d="M13 2 3 14h9l-1 8 10-12h-9l1-8Z"/>',
  dumbbell:
    '<path d="m6.5 6.5 11 11"/><path d="m21 21-1-1"/><path d="m3 3 1 1"/><path d="m18 22 4-4"/><path d="m2 6 4-4"/><path d="m3 10 7-7"/><path d="m14 21 7-7"/>',
  person:
    '<circle cx="12" cy="4.5" r="2"/><path d="M5 8.5h14"/><path d="M12 8.5V14"/><path d="m9 21 3-7 3 7"/>',
  lotus:
    '<path d="M12 4c2 2.5 2.5 5.5 0 9-2.5-3.500-2-6.500 0-9Z"/><path d="M4 9c3.500.300 6 2.300 8 6-3.500.500-6.500-1.500-8-6Z"/><path d="M20 9c-3.500.300-6 2.300-8 6 3.500.500 6.500-1.500 8-6Z"/><path d="M5 19c4 1.500 10 1.500 14 0"/>',
  timer: '<path d="M10 2h4"/><path d="M12 14l3-3"/><circle cx="12" cy="14" r="8"/>',
  racket:
    '<circle cx="9.500" cy="9.500" r="6.500"/><path d="m14.500 14.500 6 6"/><path d="M6 7h.01M9.500 6h.01M7 11h.01M11 9.500h.01M9.500 12.500h.01"/>',
  sparkles:
    '<path d="m12 3 1.900 5.100L19 10l-5.100 1.900L12 17l-1.900-5.100L5 10l5.100-1.900L12 3Z"/><path d="M19 15l.7 1.800L21.500 17.500l-1.800.7L19 20l-.7-1.800-1.800-.7 1.800-.7L19 15Z"/>',
  waves:
    '<path d="M2 8c2 0 2-1.500 4-1.500S8 8 10 8s2-1.500 4-1.500S16 8 18 8s2-1.500 4-1.500"/><path d="M2 13c2 0 2-1.500 4-1.500S8 13 10 13s2-1.500 4-1.500S16 13 18 13s2-1.500 4-1.500"/><path d="M2 18c2 0 2-1.500 4-1.500S8 18 10 18s2-1.500 4-1.500S16 18 18 18s2-1.500 4-1.500"/>',
  mountain:
    '<path d="m3 20 6.500-11 4 6.500 2-3L21 20H3Z"/><path d="M9.500 9 11 11.500l1.500-1"/><circle cx="17.500" cy="5.500" r="1.500"/>',
  pin: '<path d="M20 10c0 6-8 12-8 12S4 16 4 10a8 8 0 0 1 16 0Z"/><circle cx="12" cy="10" r="3"/>',
  verified:
    '<path d="M3.850 8.620a4 4 0 0 1 4.780-4.770 4 4 0 0 1 6.740 0 4 4 0 0 1 4.780 4.780 4 4 0 0 1 0 6.740 4 4 0 0 1-4.770 4.780 4 4 0 0 1-6.750 0 4 4 0 0 1-4.780-4.770 4 4 0 0 1 0-6.760Z"/><path d="m9 12 2 2 4-4"/>',
} as const;

export type IconName = keyof typeof PATHS;

/** Qué ícono lleva cada categoría; una categoría nueva sin ícono usa el pin. */
const CATEGORY_ICONS: Record<string, IconName> = {
  gimnasios: "dumbbell",
  pilates: "person",
  yoga: "lotus",
  entrenadores: "timer",
  padel: "racket",
  estetica: "sparkles",
  natacion: "waves",
  "trekking-outdoor": "mountain",
};

export const categoryIconName = (slug: string): IconName => CATEGORY_ICONS[slug] ?? "pin";

/** El interior del SVG, para los marcadores del mapa. */
export const iconMarkup = (name: IconName): string => PATHS[name];

export function Icon({ name, className }: { name: IconName; className?: string }) {
  return (
    <Svg className={className}>
      <g dangerouslySetInnerHTML={{ __html: PATHS[name] }} />
    </Svg>
  );
}

export const CategoryIcon = ({ slug, className }: { slug: string; className?: string }) => (
  <Icon name={categoryIconName(slug)} className={className} />
);

export const VerifiedIcon = ({ className }: { className?: string }) => (
  <Icon name="verified" className={className} />
);
