import { cn } from "@/lib/cn";

/** La etiqueta corta del mejor beneficio del local ("-15%", "-$2.000", "Beneficio"); sin ella no se dibuja nada. */
export function BenefitPill({ label, className }: { label: string | null; className?: string }) {
  if (!label) return null;
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full bg-movo-accent px-2.5 py-1 text-xs font-extrabold text-movo-ink",
        className,
      )}
    >
      {label}
    </span>
  );
}
