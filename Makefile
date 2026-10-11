.PHONY: rebuild rebuild-api rebuild-frontend

# Rebuild every image defined in docker-compose.yml.
rebuild: rebuild-api rebuild-frontend

rebuild-api:
	docker compose build api

rebuild-frontend:
	docker compose build frontend
