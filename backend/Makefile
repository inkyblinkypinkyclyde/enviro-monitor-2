.PHONY: db-setup db-teardown db-seed

db-setup:
	docker compose up -d postgres

db-teardown:
	docker compose down -v

db-seed:
	docker compose exec -T postgres sh -c 'psql -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"' < db/seed.sql
