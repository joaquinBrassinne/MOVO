import type { ReactNode } from "react";

/**
 * Marco del portal de socios: fondo, ondas decorativas y encabezado.
 * El logotipo es texto hasta que haya un archivo de marca (ver docs/work/phase-1-identity.md).
 */
export function PortalChrome({ children }: { children: ReactNode }) {
  return (
    <div className="relative isolate min-h-dvh overflow-hidden bg-movo-page font-sans text-white">
      <Waves />

      <header className="mx-auto flex w-full max-w-md items-center justify-between px-6 pt-6">
        <div className="flex items-baseline gap-4">
          <span className="font-display text-3xl font-extrabold tracking-wider text-movo-accent">
            MOVO
          </span>
          <span className="text-sm font-semibold tracking-[0.2em] text-movo-accent">SOCIOS</span>
        </div>
        <span className="inline-flex items-center gap-2 rounded-full bg-movo-tabs px-3 py-1.5 text-sm font-semibold">
          <span className="size-2 rounded-full bg-emerald-400" aria-hidden="true" />
          Portal B2B
        </span>
      </header>

      <div className="mx-auto w-full max-w-md px-4 pt-10 pb-12">{children}</div>
    </div>
  );
}

function Waves() {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 400 800"
      preserveAspectRatio="none"
      className="pointer-events-none absolute inset-0 -z-10 size-full text-movo-accent/25"
      fill="none"
      stroke="currentColor"
    >
      <path d="M-10 70 Q 100 110 200 60 T 410 50" strokeWidth="1.5" strokeDasharray="6 6" />
      <path d="M-10 105 Q 90 135 190 105 T 410 95" strokeWidth="2" />
      <path d="M-10 770 Q 120 720 230 745 T 410 700" strokeWidth="2" />
      <path d="M-10 690 Q 80 650 170 675 T 410 640" strokeWidth="1.5" strokeDasharray="6 6" />
    </svg>
  );
}
