TRUNCATE dht22_readings, sensors, rooms RESTART IDENTITY CASCADE;

INSERT INTO rooms (room_name) VALUES
    ('Living Room'),
    ('Bedroom'),
    ('Kitchen');

INSERT INTO sensors (room_id, ip_address) VALUES
    (1, '192.168.1.101'),
    (2, '192.168.1.102'),
    (3, '192.168.1.103');

-- 48 readings per sensor, one every 30 minutes covering the last 24 hours.
INSERT INTO dht22_readings (sensor_id, temperature, humidity, collected_time)
SELECT
    s.id,
    ROUND((18 + s.id + random() * 6)::numeric, 1),
    ROUND((40 + s.id * 3 + random() * 20)::numeric, 1),
    NOW() - (g.n * INTERVAL '30 minutes')
FROM sensors s
CROSS JOIN generate_series(0, 47) AS g(n);
