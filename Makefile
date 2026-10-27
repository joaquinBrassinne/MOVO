# MOVO — Plataforma de bienestar.
#
# Convención: todo target documentado con "## " aparece en `make help`.
# Las recetas asumen un shell POSIX (sh en CI/Linux). En Windows se apunta a
# Git para Windows porque make no encuentra `sh` en el PATH cuando se lo invoca
# desde PowerShell o cmd; con otra ruta, `make SHELL=/ruta/a/sh.exe`.
# Se usa la ruta corta (8.3) porque make no acepta espacios en SHELL.
ifeq ($(OS),Windows_NT)
  SHELL := $(firstword $(wildcard C:/Progra~1/Git/bin/sh.exe) sh)
else
  SHELL := sh
endif

BACKEND := backend
COMPOSE := docker compose --env-file .env -f deploy/docker-compose.yml
GO      ?= go

# Versión de golangci-lint que espera la configuración de backend/.golangci.yml
# (esquema v2) y la que corre CI. Se instala con:
#   go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
GOLANGCI_LINT_VERSION := v2.14.0
GOLANGCI_LINT         ?= golangci-lint

# Flags extra para los tests. CI agrega -race; en Windows no hay gcc (CGO off),
# así que el default local no lo pide.
TESTFLAGS ?=

.DEFAULT_GOAL := help
.PHONY: help env up down ps logs wait-db migrate migrate-up migrate-down migrate-status run-api run-worker seed-dev fmtfmt-check tidy lint test build frontend-lint frontend-test frontend-build

# --- Entorno ----------------------------------------------------------------

env: ## Crea .env desde .env.example si todavía no existe
	@test -f .env || { cp .env.example .env; echo "creado .env desde .env.example"; }

# --- Infra local -------------------------------------------------------------

up: env ## Levanta Postgres+PostGIS y RustFS, y espera los healthchecks
	$(COMPOSE) up -d --wait

down: env ## Baja los servicios conservando los volúmenes
	$(COMPOSE) down

ps: env ## Muestra el estado de los servicios
	$(COMPOSE) ps

logs: env ## Sigue los logs de los servicios
	$(COMPOSE) logs -f --tail=100

wait-db: env ## Espera (máx. 60s) a que Postgres acepte conexiones
	@i=0; until $(COMPOSE) exec -T postgres pg_isready -q >/dev/null 2>&1; do \
	  i=$$((i+1)); [ $$i -ge 60 ] && { echo "postgres no respondió en 60s"; exit 1; }; sleep 1; done; \
	echo "postgres listo"

# --- Migraciones -------------------------------------------------------------

# .env se carga porque `go run` no lo lee solo; set -a lo exporta al proceso.
# Si DATABASE_URL ya viene del entorno (CI), manda esa: el job define la suya y
# no hay archivo .env que leer.
migrate: env ## Ejecuta un comando de goose (CMD=up|down|status|version)
	@set -a; [ -n "$$DATABASE_URL" ] || . ./.env; set +a; cd $(BACKEND) && $(GO) run ./cmd/migrate $(CMD)

migrate-up: env ## Aplica las migraciones pendientes
	@$(MAKE) --no-print-directory migrate CMD=up

migrate-down: env ## Revierte la última migración aplicada
	@$(MAKE) --no-print-directory migrate CMD=down

migrate-status: env ## Muestra qué migraciones están aplicadas
	@$(MAKE) --no-print-directory migrate CMD=status

# --- Servicios ---------------------------------------------------------------

run-api: env ## Levanta la API cargando .env (go run no lo lee solo)
	@set -a; . ./.env; set +a; cd $(BACKEND) && $(GO) run ./cmd/api

run-worker: env ## Levanta el worker de tareas diarias cargando .env
	@set -a; . ./.env; set +a; cd $(BACKEND) && $(GO) run ./cmd/worker

seed-dev: env ## Carga locales de prueba (uno por categoría) en la base local
	@set -a; [ -n "$$DATABASE_URL" ] || . ./.env; set +a; cd $(BACKEND) && $(GO) run ./cmd/seed-dev

# --- Ayuda -------------------------------------------------------------------

# En Windows, awk vive en el usr/bin de Git, que no está en el PATH cuando make
# se invoca desde PowerShell o cmd: se lo agrega solo para esta receta.
help: ## Lista los targets disponibles
	@PATH="$$PATH:/usr/bin" awk 'BEGIN {FS = ":.*## "; print ""; print "MOVO - targets disponibles"; print ""} \
	     /^[a-zA-Z0-9_.-]+:.*## / {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""

# --- Código Go ---------------------------------------------------------------

fmt: ## Formatea el código Go
	cd $(BACKEND) && $(GO) fmt ./...

fmt-check: ## Falla si algún archivo Go no está formateado
	@cd $(BACKEND) && out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then echo "Archivos sin formatear:"; echo "$$out"; exit 1; fi

tidy: ## Ordena backend/go.mod y backend/go.sum
	cd $(BACKEND) && $(GO) mod tidy

lint: ## Corre golangci-lint sobre el backend
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || { \
	  echo "golangci-lint no está instalado. Instalá la versión que usa CI:"; \
	  echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)"; \
	  exit 1; }
	cd $(BACKEND) && $(GOLANGCI_LINT) run ./...

test: ## Corre los tests del backend (CI agrega TESTFLAGS="-race")
	cd $(BACKEND) && $(GO) test $(TESTFLAGS) ./...

perf-search: ## Mide la búsqueda con 10k locales falsos (necesita Docker); imprime p95 y EXPLAIN ANALYZE
	cd $(BACKEND) && $(GO) test -tags perf -run SearchLatency -v -count=1 ./internal/search/adapters/postgres

build: ## Compila los binarios del backend en backend/bin
	cd $(BACKEND) && $(GO) build -o bin/ ./cmd/...

frontend-lint: ## Corre eslint, el chequeo de tipos y el de formato del frontend
	cd frontend && npm run lint && npm run typecheck && npm run format:check

frontend-test: ## Corre los tests unitarios del frontend
	cd frontend && npm test

frontend-build: ## Compila el frontend para producción
	cd frontend && npm run build
