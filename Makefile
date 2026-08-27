.PHONY: run-api run-worker test lint build migrate-up migrate-down docker-build docker-up docker-down

# ── Development ──

run-api:
	cd backend && go run ./cmd/api

run-worker:
	cd backend && go run ./cmd/worker

test:
	cd backend && go test ./... -v -race

lint:
	cd backend && go vet ./...

build:
	cd backend && go build -o bin/api ./cmd/api
	cd backend && go build -o bin/worker ./cmd/worker

# ── Database ──

MIGRATE_URL ?= postgres://postgres:postgres@localhost:5432/jobplatform?sslmode=disable

migrate-up:
	migrate -path backend/migrations -database "$(MIGRATE_URL)" up

migrate-down:
	migrate -path backend/migrations -database "$(MIGRATE_URL)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir backend/migrations -seq $$name

# ── Docker ──

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

# ── Kubernetes ──

k8s-up:
	kubectl apply -f deploy/kubernetes/

k8s-down:
	kubectl delete -f deploy/kubernetes/

k8s-logs:
	kubectl logs -l app=jobplatform-api --tail=100 -f

# ── Combined ──

dev: docker-up run-api
