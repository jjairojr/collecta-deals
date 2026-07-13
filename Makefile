# opdeals — dev tooling
#
#   make dev     server (air live-reload) + web (Vite) together
#   make server  Go API only, air live-reload
#   make web     Vite UI only (:5173, proxies /api -> :8080)
#   make cli     run the CLI once (extra flags via ARGS="...")
#   make build   production build: web bundle + Go binaries into ./bin
#   make test    go test ./...
#
# MyP Cards is off by default; enable with MYPCARDS=1 (e.g. make dev MYPCARDS=1).

GOBIN    := $(shell go env GOPATH)/bin
AIR      := $(GOBIN)/air
MYP      := $(if $(filter 1 true yes,$(MYPCARDS)),--mypcards,)
# Pass args to the air-run binary via args_bin. -web= keeps the dev server API-only
# (Vite serves the FE); when MYPCARDS is set this overrides .air.toml's args_bin, so
# repeat -web= here too. Without MYPCARDS, .air.toml's args_bin ["-web="] applies.
AIR_ARGS := $(if $(MYP),-build.args_bin "-web= $(MYP)")
ARGS     ?=

.DEFAULT_GOAL := dev
.PHONY: dev server web cli build web-build test tidy clean ensure-air help

## dev: Go server (air) + Vite web dev, together; Ctrl-C stops both.
dev: web/node_modules ensure-air
	@echo "→ Go API (air$(if $(MYP), + mypcards,)) on :8080   +   Vite UI on http://localhost:5173"
	$(if $(MYP),@echo "  first --mypcards run scans ~6 min (headless Chrome) and caches to data/snapshot.json;")
	$(if $(MYP),@echo "  later air reloads load that cache and start instantly.")
	@trap 'kill 0' INT TERM EXIT; \
	$(AIR) $(AIR_ARGS) & \
	( cd web && npm run dev ) & \
	wait

## server: Go API only, air live-reload, --mypcards.
server: ensure-air
	$(AIR) $(AIR_ARGS)

## web: Vite dev server only.
web: web/node_modules
	cd web && npm run dev

## cli: run the CLI once with --mypcards (extra flags: make cli ARGS="--min-margin 100 --limit 20").
cli:
	go run ./cmd/opdeals $(MYP) $(ARGS)

## build: production build — web bundle + Go binaries into ./bin.
build: web-build
	go build -o ./bin/server ./cmd/server
	go build -o ./bin/opdeals ./cmd/opdeals

web-build: web/node_modules
	cd web && npm run build

test:
	go test ./...

tidy:
	go mod tidy

clean:
	rm -rf tmp bin web/dist

## ensure-air: install air if it is not present.
ensure-air:
	@test -x "$(AIR)" || { echo "installing air..."; go install github.com/air-verse/air@latest; }

web/node_modules:
	cd web && npm install

help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'
