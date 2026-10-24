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

## Portal de socios (Fase 1)

| Ruta                   | Qué es                                                                     |
| ---------------------- | -------------------------------------------------------------------------- |
| `/socios/ingresar`     | Ingreso del dueño                                                          |
| `/socios/crear-cuenta` | Alta del dueño                                                             |
| `/socios/panel`        | **Provisorio**: confirma la sesión y cierra sesión. La Fase 2 lo reemplaza |

Variables (en `frontend/.env.local`, que git ignora):

| Variable                       | Para qué                                                                                                                                                                |
| ------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `API_URL`                      | Base de la API Go que usa el servidor de Next. Por defecto `http://localhost:8080`                                                                                      |
| `NEXT_PUBLIC_SUPPORT_WHATSAPP` | Número de soporte, solo dígitos con código de país. Sin él se ocultan "¿Olvidaste tu contraseña?" y el acceso a soporte: el reseteo es asistido por WhatsApp (SDD §5.7) |

**Sesión.** El navegador solo habla con `/api/session/*` (route handler de Next, BFF). El refresh token vive en la cookie `httpOnly` `movo_refresh`, limitada a ese path, y nunca llega al JavaScript de la página; el access token queda en memoria (`lib/session`). Las renovaciones simultáneas se serializan (una sola llamada por pestaña y un lock entre pestañas), porque reutilizar un refresh token ya rotado cierra todas las sesiones.

**Verlo de punta a punta**

```bash
make up && make migrate-up                      # Postgres (si 5432 está ocupado: POSTGRES_PORT y DATABASE_URL)
cd backend && JWT_SECRET=<32+ caracteres> go run ./cmd/api   # API en :8080
cd frontend && npm run dev                       # http://localhost:3000
```

Abrir `http://localhost:3000/socios/crear-cuenta`. En desarrollo usar `localhost`, no `127.0.0.1`: Next bloquea los recursos de desarrollo desde otros orígenes y la página no hidrata.
