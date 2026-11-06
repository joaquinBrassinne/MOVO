import { CategoryIcon } from "@/components/explorar/icons";
import { cn } from "@/lib/cn";

interface PhotoImageProps {
  src: string;
  /** Vacío para una foto decorativa (la tarjeta ya dice el nombre del local). */
  alt?: string;
  loading?: "eager" | "lazy";
  className?: string;
}

/**
 * Una foto del local. Las URLs son prefirmadas y de otro origen, así que `next/image` no las
 * optimiza: este es el único lugar que usa `<img>` para ellas.
 */
export function PhotoImage({ src, alt = "", loading = "lazy", className }: PhotoImageProps) {
  // eslint-disable-next-line @next/next/no-img-element
  return <img src={src} alt={alt} loading={loading} className={cn("object-cover", className)} />;
}

interface CoverTileProps {
  /** URL de la foto de portada; `null` si el local no cargó fotos. */
  src: string | null;
  /** Sin foto se muestra el ícono de esta categoría. */
  categorySlug: string;
  /** Tamaño, forma y fondo del cuadro (`size-20 rounded-2xl bg-...`). */
  className?: string;
  iconClassName?: string;
}

/** Cuadro con la foto de portada del local, o el ícono de su categoría si no tiene. */
export function CoverTile({ src, categorySlug, className, iconClassName }: CoverTileProps) {
  return (
    <span
      className={cn(
        "relative grid shrink-0 place-items-center overflow-hidden text-movo-accent",
        className,
      )}
    >
      {src ? (
        <PhotoImage src={src} className="absolute inset-0 size-full" />
      ) : (
        <CategoryIcon slug={categorySlug} className={iconClassName} />
      )}
    </span>
  );
}
