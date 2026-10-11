# Backend

## Run

```sh
make db-setup   # start Postgres and create tables
make db-seed    # optional dummy data
cd api && go run .
```

The API listens on `http://localhost:8080` (override with `PORT`).

## Sample requests

```sh
# Health check (503 if the database is unreachable)
curl -i http://localhost:8080/health

# All rooms
curl http://localhost:8080/rooms

# All sensors
curl http://localhost:8080/sensors

# Latest 100 readings across all sensors
curl http://localhost:8080/readings

# Latest readings for one sensor
curl "http://localhost:8080/readings?sensor_id=1"

# Invalid sensor_id returns 400
curl -i "http://localhost:8080/readings?sensor_id=abc"
```

Pipe to `jq` for readable output, e.g. `curl -s http://localhost:8080/rooms | jq`.

## Tests

```sh
make test       # handler unit tests, no database needed
make test-all   # handlers + store tests; requires the database and resets its tables
```
