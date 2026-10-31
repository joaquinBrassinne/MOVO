export default function Loading() {
  return (
    <div
      className="min-h-dvh animate-pulse bg-movo-cream"
      aria-busy="true"
      aria-label="Cargando el local"
    >
      <div className="h-72 bg-movo-card" />
      <div className="space-y-4 px-4 pt-8">
        <div className="h-8 w-2/3 rounded-full bg-movo-sand" />
        <div className="h-14 rounded-full bg-movo-sand" />
        <div className="h-32 rounded-3xl bg-movo-sand" />
      </div>
    </div>
  );
}
