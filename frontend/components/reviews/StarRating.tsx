import { cn } from "@/lib/cn";
import { formatRating, isNewBranch, starFills } from "@/lib/reviews/format";

const STAR_PATH =
  "M12 2.8a.6.6 0 0 1 1 0l2.6 5.6 6.1.8a.6.6 0 0 1 .3 1l-4.5 4.3 1.1 6a.6.6 0 0 1-.9.7L12 18.3l-5.4 2.9a.6.6 0 0 1-.9-.7l1.1-6L2.3 10.2a.6.6 0 0 1 .3-1l6.1-.8Z";

/** Una estrella gris con la parte llena (0, ½ o 1) encima. */
function Star({ fill, className }: { fill: number; className: string }) {
  return (
    <span className={cn("relative inline-block", className)} aria-hidden="true">
      <svg
        viewBox="0 0 24 24"
        className="absolute inset-0 size-full fill-current text-movo-slate/40"
      >
        <path d={STAR_PATH} />
      </svg>
      <span
        className="absolute inset-y-0 left-0 overflow-hidden"
        style={{ width: `${fill * 100}%` }}
      >
        <svg viewBox="0 0 24 24" className={cn("fill-current text-movo-accent", className)}>
          <path d={STAR_PATH} />
        </svg>
      </span>
    </span>
  );
}

/**
 * Promedio en estrellas (llenas, medias y vacías) con el número al lado. Con menos de 3
 * reseñas muestra "Nuevo" en vez de estrellas.
 */
export function StarRating({
  average,
  count,
  showCount = false,
  onDark = false,
  starClassName = "size-4",
  className,
}: {
  average: number;
  count: number;
  showCount?: boolean;
  /** Sobre las tarjetas oscuras del inicio: el "Nuevo" usa el dorado claro. */
  onDark?: boolean;
  starClassName?: string;
  className?: string;
}) {
  if (isNewBranch(count)) {
    return (
      <span
        className={cn(
          "inline-flex items-center rounded-full bg-movo-accent/30 px-2.5 py-1 text-xs font-bold",
          onDark ? "text-movo-accent" : "text-movo-gold",
          className,
        )}
      >
        Nuevo
      </span>
    );
  }

  const label = formatRating(average);
  return (
    <span
      role="img"
      aria-label={`${label} de 5 estrellas, ${count} reseñas`}
      className={cn("inline-flex items-center gap-1.5 text-sm font-bold", className)}
    >
      <span className="inline-flex gap-0.5">
        {starFills(average).map((fill, index) => (
          <Star key={index} fill={fill} className={starClassName} />
        ))}
      </span>
      <span aria-hidden="true">{label}</span>
      {showCount ? (
        <span aria-hidden="true" className="font-normal opacity-70">
          ({count})
        </span>
      ) : null}
    </span>
  );
}
