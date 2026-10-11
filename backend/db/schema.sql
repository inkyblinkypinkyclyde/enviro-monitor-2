CREATE TABLE IF NOT EXISTS rooms (
    id SERIAL PRIMARY KEY,
    room_name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sensors (
    id SERIAL PRIMARY KEY,
    room_id INT NOT NULL REFERENCES rooms (id)
);

CREATE TABLE IF NOT EXISTS dht22_readings (
    id SERIAL PRIMARY KEY,
    sensor_id INT NOT NULL REFERENCES sensors (id),
    temperature REAL NOT NULL,
    humidity REAL NOT NULL,
    collected_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
