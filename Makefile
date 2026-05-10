ifneq (,$(wildcard ./.env))
    include .env
    export
endif

templ:
	@templ generate

dev: templ
	@go run ./cmd/app/

build: templ
	@go build -o bin/dashboard ./cmd/app/
	@echo "built to bin/dashboard"

.PHONY: templ dev build
