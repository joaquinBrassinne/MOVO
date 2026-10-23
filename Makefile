# MOVO — Plataforma de bienestar.
#
# Convención: todo target documentado con "## " aparece en `make help`.
# Las recetas asumen un shell POSIX (Git Bash en Windows, sh en CI/Linux).

SHELL := sh

BACKEND := backend
COMPOSE := docker compose --env-file .env -f deploy/docker-compose.yml
GO      ?= go

.DEFAULT_GOAL := help
.PHONY: help env up down ps logs wait-db fmt fmt-check tidy

# --- Entorno ----------------------------------------------------------------

env: ## Crea .env desde .env.example si todavía no existe
	@test -f .env || { cp .env.example .env; echo "creado .env desde .env.example"; }

# --- Infra local -------------------------------------------------------------

up: env ## Levanta Postgres+PostGIS y MinIO, y espera los healthchecks
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

# --- Ayuda -------------------------------------------------------------------

help: ## Lista los targets disponibles
	@awk 'BEGIN {FS = ":.*## "; print ""; print "MOVO - targets disponibles"; print ""} \
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
