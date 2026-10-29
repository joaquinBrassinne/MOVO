import { cn } from "@/lib/cn";
import type { ReviewFilter } from "@/lib/reviews/useBranchReviews";

/** The star chips of the public list. Only `rating` and `min_rating` exist in the API. */
export function ReviewFilterChips({
  filter,
  total,
  onChange,
}: {
  filter: ReviewFilter;
  total: number;
  onChange: (filter: ReviewFilter) => void;
}) {
  const chips: { value: ReviewFilter; label: string }[] = [
    { value: "all", label: `Todas (${total})` },
    { value: "5", label: "5 estrellas" },
    { value: "4plus", label: "4 estrellas o más" },
  ];

  return (
    <div role="group" aria-label="Filtrar por estrellas" className="flex flex-wrap gap-2">
      {chips.map((chip) => (
        <button
          key={chip.value}
          type="button"
          aria-pressed={filter === chip.value}
          onClick={() => onChange(chip.value)}
          className={cn(
            "h-11 rounded-full px-4 text-sm font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink",
            filter === chip.value ? "bg-movo-page text-white" : "bg-white text-movo-ink shadow-sm",
          )}
        >
          {chip.label}
        </button>
      ))}
    </div>
  );
}
