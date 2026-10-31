"use client";

export default function ProfileError({ reset }: { reset: () => void }) {
  return (
    <main className="grid min-h-dvh place-items-center bg-movo-cream px-6 text-center text-movo-ink">
      <div>
        <h1 className="font-display text-2xl font-extrabold">No pudimos cargar el local</h1>
        <p className="mt-2 text-movo-slate">Probá de nuevo en un rato.</p>
        <button
          type="button"
          onClick={reset}
          className="mt-5 h-12 rounded-full bg-movo-accent px-6 font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
        >
          Reintentar
        </button>
      </div>
    </main>
  );
}
