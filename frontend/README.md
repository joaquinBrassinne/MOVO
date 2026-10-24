# Frontend MOVO

Next.js (App Router) + Tailwind CSS v4, en TypeScript estricto.

## Comandos

```bash
npm install          # dependencias
npm run dev          # servidor de desarrollo en http://localhost:3000
npm run lint         # ESLint
npm run typecheck    # next typegen + tsc --noEmit
npm run format       # Prettier (escribe)
npm run format:check # Prettier (verifica)
npm run build        # build de producción
npm run start        # sirve el build
```

`typecheck` corre `next typegen` primero porque los tipos de ruta (`LayoutProps`, `PageProps`) los genera Next.

## Convenciones

- Lógica de negocio y llamadas a la API en hooks, no en componentes.
- SSR para páginas públicas; los mapas van en componentes cliente.
- Versiones exactas en `package.json` (sin `^`): las actualiza una tarea, no el azar.
- Reglas del proyecto: ver `../CLAUDE.md`.
