export default function SitioLayout({ children }: LayoutProps<"/">) {
  return (
    <div className="flex min-h-dvh flex-col">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex w-full max-w-5xl items-center justify-between px-6 py-4">
          <span className="text-lg font-semibold tracking-tight text-brand-700">MOVO</span>
          <nav aria-label="Principal" className="text-sm text-slate-600">
            <span>Bienestar cerca tuyo</span>
          </nav>
        </div>
      </header>
      <div className="flex-1">{children}</div>
      <footer className="border-t border-slate-200 bg-white">
        <div className="mx-auto w-full max-w-5xl px-6 py-4 text-sm text-slate-500">
          MOVO · Almafuerte y Río Tercero, Córdoba
        </div>
      </footer>
    </div>
  );
}
