.PHONY: db-setup db-teardown db-seed test test-all

db-setup:
	docker compose up -d postgres

db-teardown:
	docker compose down -v

db-seed:
	docker compose exec -T postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"' < db/seed.sql

# Unit tests only; no database needed.
test:
	cd api && go test ./handlers/... ./collector/...

# Handler tests plus store integration tests; needs the database from db-setup.
test-all:
	cd api && go test -tags integration ./...
