import Image from "next/image";

interface LogoProps {
  /** Clase de altura (Tailwind); el ancho sale de la proporción 3:1 del logo. */
  className?: string;
}

/** Wordmark de MOVO en dorado: pensado para fondos oscuros. */
export function Logo({ className = "h-8" }: LogoProps) {
  return (
    <Image
      src="/movo-logo.png"
      alt="MOVO"
      width={2000}
      height={667}
      priority
      className={`w-auto ${className}`}
    />
  );
}
