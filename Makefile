MIGRATION_PATH = ./migrations
NAME ?= default_migration

.PHONY: build
build:
	CONFIG_PATH=./configs/local.yaml go build -o ./bin ./cmd/social/main.go

.PHONY: watch
watch:
	CONFIG_PATH=./config/local.yaml air

.PHONY: start-db
start-db:
	docker compose up --build

.PHONY: migrate-create
migrate-create:
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate create -ext sql -dir migrations $(NAME)

.PHONY: migrate-up
migrate-up:
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate \
        -path $(MIGRATION_PATH) \
        -database "postgresql://bobahan:pass@localhost:5432/social?sslmode=disable" \
        up

.PHONY: migrate-down
migrate-down:
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate \
        -path $(MIGRATION_PATH) \
        -database "postgresql://bobahan:pass@localhost:5432/social?sslmode=disable" \
        down

.PHONY: revert-last_migration
revert-last_migration:
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate \
        -path $(MIGRATION_PATH) \
        -database "postgresql://bobahan:pass@localhost:5432/social?sslmode=disable" \
        down 1

.PHONY: seed-database
seed-database:
	CONFIG_PATH=./config/local.yaml go run ./migrations/seed/main.go

.PHONY: swag
swag:
	swag init -g ./cmd/social/main.go && swag fmt