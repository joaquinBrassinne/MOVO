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
