const categories = ["Gimnasios", "Pilates", "Yoga", "Entrenadores", "Estética", "Piletas"];

export default function Home() {
  return (
    <main className="mx-auto w-full max-w-5xl px-6 py-16">
      <h1 className="max-w-2xl text-4xl font-semibold tracking-tight text-slate-900">
        Encontrá tu actividad y canjeá beneficios en el local.
      </h1>
      <p className="mt-4 max-w-2xl text-lg text-slate-600">
        Gimnasios, pilates, yoga, entrenadores, estética y piletas de tu ciudad, con beneficios que
        se canjean en el lugar.
      </p>
      <ul className="mt-10 flex flex-wrap gap-2">
        {categories.map((category) => (
          <li
            key={category}
            className="rounded-full bg-brand-100 px-4 py-1.5 text-sm font-medium text-brand-700"
          >
            {category}
          </li>
        ))}
      </ul>
    </main>
  );
}
