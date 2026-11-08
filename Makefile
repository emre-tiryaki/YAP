SHELL := /bin/bash

-include .env
BACKEND_PORT ?= 8080

.PHONY: dev

dev:
	@trap 'kill 0' EXIT INT TERM; \
	(cd backend && exec go run ./cmd/backend) & BACK=$$!; \
	until curl -sf http://localhost:$(BACKEND_PORT)/hello-world >/dev/null; do \
		kill -0 $$BACK 2>/dev/null || { echo "backend failed to start"; exit 1; }; \
		sleep 0.5; \
	done; \
	(cd apps/web && exec bun run dev) & WEB=$$!; \
	wait -n $$BACK $$WEB