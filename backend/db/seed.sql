TRUNCATE dht22_readings, sensors, rooms RESTART IDENTITY CASCADE;

INSERT INTO rooms (room_name) VALUES
    ('Living Room'),
    ('Bedroom'),
    ('Kitchen');

INSERT INTO sensors (room_id) VALUES
    (1),
    (2),
    (3);

-- 10 readings per sensor, one every 30 minutes ending now.
INSERT INTO dht22_readings (sensor_id, temperature, humidity, collected_time)
SELECT
    s.id,
    ROUND((18 + s.id + random() * 6)::numeric, 1),
    ROUND((40 + s.id * 3 + random() * 20)::numeric, 1),
    NOW() - (g.n * INTERVAL '30 minutes')
FROM sensors s
CROSS JOIN generate_series(0, 9) AS g(n);
