# Fase 0 — Tarea 0.8: frontend Next.js + Tailwind base

Rama `phase-0/foundations`. Entregable: un frontend que arranca, compila y pasa lint y formato, con el layout base listo para las páginas de la Fase 1. Sin features del dominio.

## Specs

- **S1** (§2, tabla de stack): `| Frontend | Next.js (App Router) + Tailwind | SSR para SEO de fichas de locales |`
- **S2** (§9, checklist): `- [ ] 0.8 Frontend Next.js + Tailwind base, ESLint/Prettier, layout (S)`
- **S3** (DoD de la fase): `**DoD:** make up levanta todo; CI verde; /readyz responde con la DB conectada.`
- **S4** (`CLAUDE.md`, reglas de frontend):
  - `Componentes limpios; lógica de negocio y llamadas a la API en hooks personalizados.`
  - `SSR para páginas públicas (fichas de locales); mapas solo en componentes cliente.`
  - `Tipos compartidos desde el contrato OpenAPI cuando sea posible.`
- **S5** (`CLAUDE.md`, stack): `Next.js (App Router) + Tailwind. Mapas: OpenStreetMap + React Leaflet.`
- **S6** (§7.1): `- Dependencias auditadas (govulncheck, npm audit) en CI.`

> **S6 fuera de alcance:** `npm audit` en CI entra en la tarea **8.1**, igual que `govulncheck` (decisión tomada y registrada en la tarea 0.6).

## Decisiones de versiones (registro al 2026-10-23, antigüedad ≥ 2 semanas)

| Paquete | Versión | Publicada | Descartada por edad |
|---|---|---|---|
| `next` (+ `eslint-config-next`) | **16.4.2** | 2026-10-15 | 16.4.0 por advisory crítico (ver L5) |
| `react` / `react-dom` | **19.3.0** | 2026-09-09 | 19.3.1 (10-14) |
| `eslint` | **10.12.0** | 2026-10-02 | 10.13.0 (10-16) |
| `eslint-config-prettier` | **10.1.8** | 2025-07-18 | — |
| `prettier` | **3.9.9** | 2026-09-23 | 3.9.10 (10-10) |
| `tailwindcss` + `@tailwindcss/turbopack` | **4.3.3** | 2026-07-16 | — |
| `typescript` | **6.0.3** | 2026-04-16 | 7.0.2 y 6.0.3 quedan fuera de rango: ver L6 |
| `@types/react`, `@types/react-dom` | **19.3.0** | 2026-09-09 | — |
| `@types/node` | **26.6.5** | 2026-10-09 | — (14 días exactos) |
| `actions/setup-node` (CI) | **v7.1.0** | 2026-10-08 | — (15 días) |

> **TS 7** es la reescritura nativa del compilador. Se descartó por contrato de peer, no por edad: ver L6.

El scaffold usa **`@tailwindcss/turbopack`** (loader de Turbopack), no PostCSS: es lo que genera `create-next-app` en esta versión y no hace falta agregar el plugin de PostCSS por encima.

## Tasks

| ID | Tarea | Specs | Commit |
|----|-------|-------|--------|
| T1 | Scaffold `frontend/` con `create-next-app@16.4.0` (App Router, TS, Tailwind, ESLint, sin `src/`), versiones pineadas | S1, S2 | `1b8ff2d` (+ `5e7d958` lockfile) |
| T2 | ESLint flat config + Prettier + scripts `lint`/`typecheck`/`format`/`format:check` | S2, S4 | `1b8ff2d` |
| T3 | Layout base con Tailwind y copy en español; `globals.css` con el tema | S1, S4 | `1b8ff2d` |
| T4 | Job `frontend` en CI (lint + typecheck + build) y su test de contrato | S3 | `dd6a862` |
| T5 | Docs: checkbox 0.8 y comandos del frontend en `CLAUDE.md` | S2, S3 | `dd6a862` |
| T6 | Verificación: `npm run lint`, `format:check`, `typecheck`, `build` en local + run de CI verde | S3 | run `39896206545` |

## Log

- **L1** (pedido original, verbatim del SDD): construir la plataforma MOVO por fases, una tarea del SDD por vez.
- **L2** (decisiones previas aprobadas): TypeScript en modo estricto, Tailwind v4, npm como gestor, frontend en el host (el compose sólo levanta infraestructura).
- **L3** (decisiones de esta tarea): sin directorio `src/` (el SDD no lo pide y evita un nivel de anidamiento); copy de la UI en **español**, porque el producto es para locales de Argentina y los usuarios finales son hispanohablantes — la regla de artefactos en inglés cede ante el idioma del producto, no al revés.
- **L4** (riesgo): T4 toca el pipeline de delivery (alto riesgo, item 5). El run real de CI es la prueba, no la validación local.
- **L5** (decisión del usuario, verbatim): el usuario eligió *"Usar next 16.4.2 (recomendado)"* frente a la alternativa de mantener 16.4.0. Motivo: `npm audit` marca en 16.4.0 un **RCE crítico** (Image Optimization API con SVG), otro RCE en `use cache` y un DoS en Server Actions; el rango vulnerable es `16.0.0 - 16.4.0`, así que **ninguna versión con ≥2 semanas estaba limpia**. 16.4.2 tiene 8 días: excepción explícita a la regla de antigüedad, por seguridad. Quedan 5 advisories `high` de la cadena de globs (`braces`/`micromatch`/`fast-glob`) que son **sólo de desarrollo** y cuyo único "fix" que ofrece npm es bajar `eslint-config-next` a 14.2.35.
- **L6** (TS 6.0.3, no 7.0.2): `eslint-config-next` depende de `typescript-eslint`, que declara `typescript: >=4.8.4 <6.1.0`. Ni TS 7 ni TS 6.0.3+ entran en ese rango; dentro de la línea soportada, la última es 6.0.3 (2026-04-16). Decidir la versión del compilador por su peer real es más honesto que forzar la última y silenciar el warning.
- **L7** (bug propio, verificado): escribí `eslint.config.mjs` con comentarios `#`, sintaxis de YAML. `eslint`, `prettier` y `npm run format:check` fallaban con `SyntaxError`; corregido a `//`.
- **L8** (`typecheck` = `next typegen && tsc --noEmit`): los tipos `LayoutProps`/`PageProps` los genera Next en `.next/types`, que está gitignoreado, así que `tsc --noEmit` solo falla en un clon limpio (`Cannot find name 'LayoutProps'`). El script corre `next typegen` primero; verificado en local y en CI.
- **L9** (test refinado, no debilitado): `TestWorkflowKeepsMakeAtTheRepositoryRoot` prohibía **cualquier** `defaults:`. El job `frontend` necesita un `defaults` de job (no usa `make`), así que la regla pasa a prohibir sólo el `defaults` de nivel superior, que es el defecto que se pagó. Sigue prohibiendo el `working-directory: backend` global.
- **L10** (verificación independiente, `gentle-ai-verify`, 2026-10-23): **PASS**. `npm run lint`, `format:check`, `typecheck` y `build` en `EXIT=0` (rutas `/` y `/_not-found` estáticas); `make lint` → `0 issues`; `make test` → `ok`, incluido `internal/ciworkflow`. Run de CI `39896206545` sobre el SHA de este doc (`b1a3112`): los cinco jobs (`migrations`, `tests`, `build`, `lint`, `frontend`) en `success`. Sondas independientes en un directorio temporal: `defaults:` de nivel superior, `working-directory: backend` global y el borrado del paso `format:check` rompen el test de contrato; un error de tipos rompe `typecheck` y `build`; un archivo sin formatear rompe `format:check`. S1–S5 cumplidas, S6 fuera de alcance.
- **L11** (desvío aceptado, observado por el verificador): T1, T2 y T3 quedaron en un solo commit (`1b8ff2d`, 254 líneas) en vez de tres; la tabla de tareas ya registra el commit real de cada una. Ninguna spec pide un commit por tarea y el tamaño quedó bajo el límite de 400 líneas.
- **L12** (sugerencia descartada, con motivo): el verificador propuso un test que fije qué scripts declara `frontend/package.json`. No se agrega: si un script falta, el propio job `frontend` falla en el run real, que es la prueba que pide S3; el test de contrato existe para no esperar el run cuando cambia el workflow, no para duplicar lo que el run ya cubre.
- **L13** (DoD de la fase, verificado el 2026-10-23 en esta rama): `make up` → `movo-postgres-1` y `movo-rustfs-1` en `healthy`; `make migrate-status` → `00001_extensions.sql` aplicada; `go run ./cmd/api` con el `.env` → `GET /healthz` `200 {"status":"ok"}` y `GET /readyz` `200 {"status":"ready"}` (base conectada), ambos con `X-Request-Id`. CI del HEAD en verde. Con eso la Fase 0 queda cerrada: 0.1–0.8 en `[x]` en `docs/sdd.md` y su DoD observado.
