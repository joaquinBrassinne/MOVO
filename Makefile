# MOVO — Plataforma de bienestar.
#
# Convención: todo target documentado con "## " aparece en `make help`.
# Las recetas asumen un shell POSIX (Git Bash en Windows, sh en CI/Linux).

SHELL := sh

BACKEND := backend
GO      ?= go

.DEFAULT_GOAL := help
.PHONY: help fmt fmt-check tidy

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
