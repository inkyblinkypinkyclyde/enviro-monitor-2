//go:build integration

package store_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"enviro-monitor/api/store"

	"github.com/uptrace/bun/driver/pgdriver"
)

func dsn() string {
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		return v
	}
	return "postgres://postgres:postgres@localhost:5432/enviro_monitor?sslmode=disable"
}

// newStore resets the tables to a known fixture and returns a Store on the same database.
func newStore(t *testing.T) *store.Store {
	t.Helper()

	raw := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn())))
	t.Cleanup(func() { raw.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stmts := []string{
		`TRUNCATE dht22_readings, sensors, rooms RESTART IDENTITY CASCADE`,
		`INSERT INTO rooms (room_name) VALUES ('Living Room'), ('Bedroom')`,
		`INSERT INTO sensors (room_id) VALUES (1), (2)`,
		`INSERT INTO dht22_readings (sensor_id, temperature, humidity, collected_time) VALUES
			(1, 20.0, 40.0, NOW() - INTERVAL '2 hours'),
			(1, 21.0, 41.0, NOW() - INTERVAL '1 hour'),
			(2, 18.5, 55.0, NOW())`,
	}
	for _, s := range stmts {
		if _, err := raw.ExecContext(ctx, s); err != nil {
			t.Fatalf("setup %q: %v", s, err)
		}
	}

	st := store.Open(dsn())
	t.Cleanup(func() { st.Close() })
	return st
}

func TestPing(t *testing.T) {
	st := newStore(t)
	if err := st.Ping(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestRooms(t *testing.T) {
	st := newStore(t)
	rooms, err := st.Rooms(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rooms) != 2 || rooms[0].RoomName != "Living Room" || rooms[1].RoomName != "Bedroom" {
		t.Fatalf("got %+v", rooms)
	}
	for i, wantTemp := range []float32{21.0, 18.5} {
		if len(rooms[i].Sensors) != 1 {
			t.Fatalf("room %d sensors = %+v", i, rooms[i].Sensors)
		}
		lr := rooms[i].Sensors[0].LatestReading
		if lr == nil || lr.Temperature != wantTemp {
			t.Errorf("room %d latest reading = %+v, want temp %v", i, lr, wantTemp)
		}
	}
}

func TestSensors(t *testing.T) {
	st := newStore(t)
	sensors, err := st.Sensors(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(sensors) != 2 || sensors[0].RoomID != 1 || sensors[1].RoomID != 2 {
		t.Errorf("got %+v", sensors)
	}
}

func TestReadingsAll(t *testing.T) {
	st := newStore(t)
	got, err := st.Readings(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	// Newest first.
	if got[0].SensorID != 2 {
		t.Errorf("first reading sensor = %d, want 2", got[0].SensorID)
	}
	for i := 1; i < len(got); i++ {
		if got[i].CollectedTime.After(got[i-1].CollectedTime) {
			t.Errorf("readings not ordered newest first at %d", i)
		}
	}
}

func TestReadingsBySensor(t *testing.T) {
	st := newStore(t)
	id := 1
	got, err := st.Readings(context.Background(), &id)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	for _, r := range got {
		if r.SensorID != 1 {
			t.Errorf("sensor = %d, want 1", r.SensorID)
		}
	}
	if got[0].Temperature != 21.0 || got[0].Humidity != 41.0 {
		t.Errorf("newest = %+v", got[0])
	}
}

func TestReadingsUnknownSensor(t *testing.T) {
	st := newStore(t)
	id := 999
	got, err := st.Readings(context.Background(), &id)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("got %#v, want empty non-nil slice", got)
	}
}
