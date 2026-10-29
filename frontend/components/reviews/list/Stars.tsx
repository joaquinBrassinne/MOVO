import { cn } from "@/lib/cn";
import { starFills } from "@/lib/reviews/format";

const STAR_PATH =
  "M12 2.8a.6.6 0 0 1 1 0l2.6 5.6 6.1.8a.6.6 0 0 1 .3 1l-4.5 4.3 1.1 6a.6.6 0 0 1-.9.7L12 18.3l-5.4 2.9a.6.6 0 0 1-.9-.7l1.1-6L2.3 10.2a.6.6 0 0 1 .3-1l6.1-.8Z";

/** Five stars filled up to `value` (halves allowed), without the number next to them. */
export function Stars({
  value,
  label,
  className = "size-4",
}: {
  value: number;
  label: string;
  className?: string;
}) {
  return (
    <span role="img" aria-label={label} className="inline-flex gap-0.5">
      {starFills(value).map((fill, index) => (
        <span key={index} aria-hidden="true" className={cn("relative inline-block", className)}>
          <svg
            viewBox="0 0 24 24"
            className="absolute inset-0 size-full fill-current text-movo-slate/30"
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
      ))}
    </span>
  );
}
