import Link from "next/link";

export default function ProfileNotFound() {
  return (
    <main className="grid min-h-dvh place-items-center bg-movo-cream px-6 text-center text-movo-ink">
      <div>
        <h1 className="font-display text-2xl font-extrabold">No encontramos ese local</h1>
        <p className="mt-2 text-movo-slate">Puede que ya no esté en MOVO.</p>
        <Link
          href="/map"
          className="mt-5 inline-flex h-12 items-center rounded-full bg-movo-accent px-6 font-bold outline-none focus-visible:ring-2 focus-visible:ring-movo-ink"
        >
          Buscar en el mapa
        </Link>
      </div>
    </main>
  );
}
