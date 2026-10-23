# Fase 0 — Tarea 0.6: CI (lint + test + build + chequeo de migraciones)

Rama `phase-0/foundations`. Un solo entregable: el pipeline de integración continua que valida el backend Go en cada push y PR contra `develop`.

## Specs

- **S1** (§2, tabla de stack): `| CI | GitHub Actions | lint, tests, build, chequeo de migraciones |`
- **S2** (§9, checklist): `- [ ] 0.6 CI (lint + test + build + chequeo de migraciones) (M)`
- **S3** (§7.4): `- golangci-lint (incl. errcheck, govet, staticcheck, depguard).`
- **S4** (§7.4): `Sin variables globales mutables; dependencias inyectadas por constructor (main.go es la raíz de composición).`
- **S5** (DoD de la fase): `**DoD:** make up levanta todo; CI verde; /readyz responde con la DB conectada.`
- **S6** (§7.1): `- Dependencias auditadas (govulncheck, npm audit) en CI.`
- **S7** (§9, checklist): `- [ ] 8.1 Checklist de seguridad (sección 7.1) y govulncheck/npm audit en CI (M)`

> **Resolución S6 vs S7:** `govulncheck` y `npm audit` entran en la tarea **8.1**, no en 0.6. El alcance de 0.6 es el que fija S2. Queda registrado acá para no perderlo.

- **S8** (`CLAUDE.md`, comandos): `- make lint / make test / make test-integration`
  El pipeline invoca los mismos targets que el desarrollador, para que "verde en local" y "verde en CI" signifiquen lo mismo.
- **S9** (§7.4): `- Errores siempre manejados y envueltos (fmt.Errorf("...: %w", err)); errores de dominio tipados.`

## Tasks

| ID | Tarea | Specs | Ruta | Commit |
|----|-------|-------|------|--------|
| T1 | Targets `lint`, `test`, `build` en el `Makefile` (mismos comandos en local y CI) | S8 | inline | `chore(make): …` |
| T2 | `.golangci.yml` con el esquema v2 y el set de S3 | S3, S9 | inline | `chore(lint): …` |
| T3 | `.github/workflows/ci.yml`: jobs `lint`, `test`, `build`, `migrations` (PostgreSQL+PostGIS como servicio) | S1, S2 | inline | `ci: …` |
| T4 | Docs: checkbox 0.6 y comandos de `CLAUDE.md` si cambian | S2, S5 | inline | junto a T3 |
| T5 | Verificación: lint/test/build en local, job de migraciones simulado contra la DB de compose, y **run real de GitHub Actions en verde** (`gh run watch`) | S5 | inline | — |

## Log

- **L1** (pedido del usuario, verbatim): *"pasame el comando para instalar gh"* → instalado `gh` 2.103.0, `gh auth status` en verde con scopes `repo` y `workflow`.
- **L2** (pedido original de la fase, verbatim del SDD): construir la plataforma MOVO por fases, una tarea del SDD por vez.
- **L3** (decisión, versiones por antigüedad ≥2 semanas al 2026-10-23): `golangci/golangci-lint v2.15.0` (2026-10-16) **descartada** por 7 días → `v2.14.0` (2026-09-24). `golangci/golangci-lint-action v9.3.0` (2026-06-29). `actions/checkout v7.0.1` y `actions/setup-go v7.0.0` (julio 2026).
- **L4** (decisión): el runner lo fija S1 (GitHub Actions); no se consultó porque el SDD ya lo decide.
- **L5** (corrección de un default previo): «`depguard` vacío» no es viable en el esquema v2 — sin `rules`, depguard sólo permite `$gostd` y marcaría en rojo `pgx`, `chi` y `goose` que ya usamos. Se resuelve en T2 y se documenta en el archivo.
