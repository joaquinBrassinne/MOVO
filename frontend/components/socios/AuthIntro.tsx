import { StoreIcon } from "./icons";

/** Título y bajada de las vistas de acceso; el ingreso suma la etiqueta "Acceso socios". */
export function AuthIntro({ badge = false }: { badge?: boolean }) {
  return (
    <div className="flex flex-col items-center text-center">
      {badge ? (
        <span className="mb-4 inline-flex items-center gap-2 rounded-full bg-movo-tabs px-4 py-2 text-sm font-bold tracking-wider text-movo-accent uppercase">
          <StoreIcon className="size-4" />
          Acceso socios
        </span>
      ) : null}
      <h1 className="font-display text-[1.75rem] leading-tight font-extrabold tracking-tight">
        Tu negocio en MOVO
      </h1>
      <p className="mt-3 max-w-xs text-base text-movo-muted">
        Para dueños de gimnasios, estudios, clubes y profesionales.
      </p>
    </div>
  );
}
