.PHONY: help db-up api worker test-api build-api build-worker smoke ci

help:
	@printf '%s\n' \
		'make db-up         # levanta PostgreSQL con docker compose' \
		'make api           # arranca la API' \
		'make worker        # arranca el event processor' \
		'make test-api      # corre tests del API' \
		'make build-api     # compila el API' \
		'make build-worker  # compila el worker' \
		'make smoke         # ejecuta el smoke test end-to-end' \
		'make ci            # corre el flujo base de CI local'

db-up:
	docker compose up -d db

api:
	cd cmd/api && go run .

worker:
	cd cmd/event-processor && go run .

test-api:
	cd cmd/api && go test ./...

build-api:
	cd cmd/api && go build ./...

build-worker:
	cd cmd/event-processor && go build ./...

smoke:
	bash ./scripts/smoke_test_pipeline.sh

ci: test-api build-api build-worker smoke
