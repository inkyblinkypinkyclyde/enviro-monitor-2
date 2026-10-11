package collector_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"enviro-monitor/api/collector"
	"enviro-monitor/api/models"
)

type fakeLister struct {
	sensors []models.Sensor
	err     error
}

func (f fakeLister) Sensors(context.Context) ([]models.Sensor, error) { return f.sensors, f.err }

// sensorServer starts a fake sensor and returns a collector pointed at it.
func sensorServer(t *testing.T, h http.HandlerFunc, sensors ...models.Sensor) *collector.Collector {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatal(err)
	}

	c := collector.New(fakeLister{sensors: sensors}, srv.Client())
	c.Port = port
	return c
}

func localSensor(id int) models.Sensor {
	return models.Sensor{ID: id, RoomID: id, IPAddress: "127.0.0.1"}
}

func TestGetReading(t *testing.T) {
	var gotPath string
	c := sensorServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"hostname":"lounge","temperature":21.5,"humidity":48.25}`))
	})

	got, err := c.GetReading(context.Background(), localSensor(3))
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/status" {
		t.Errorf("path = %q, want /status", gotPath)
	}
	if got.SensorID != 3 || got.Temperature != 21.5 || got.Humidity != 48.25 {
		t.Errorf("got %+v", got)
	}
	if got.CollectedTime.IsZero() {
		t.Error("CollectedTime not set")
	}
}

func TestGetReadingErrors(t *testing.T) {
	cases := map[string]http.HandlerFunc{
		"non-200": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "nope", http.StatusInternalServerError)
		},
		"invalid json": func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"temperature":nan}`))
		},
		"missing humidity": func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"temperature":20}`))
		},
	}
	for name, h := range cases {
		t.Run(name, func(t *testing.T) {
			c := sensorServer(t, h)
			if _, err := c.GetReading(context.Background(), localSensor(1)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestGetReadingsPollsEverySensor(t *testing.T) {
	c := sensorServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"hostname":"x","temperature":20,"humidity":50}`))
	}, localSensor(1), localSensor(2), localSensor(3))

	results, err := c.GetReadings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("len = %d, want 3", len(results))
	}
	for i, r := range results {
		if r.Err != nil || r.Reading == nil {
			t.Fatalf("result %d: %+v", i, r)
		}
		if r.Sensor.ID != i+1 || r.Reading.SensorID != i+1 {
			t.Errorf("result %d matched to wrong sensor: %+v", i, r)
		}
	}
}

func TestGetReadingsOneSensorFailing(t *testing.T) {
	c := sensorServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"hostname":"x","temperature":20,"humidity":50}`))
	}, localSensor(1),
		// Nothing listens on this loopback address, so the request fails.
		models.Sensor{ID: 2, RoomID: 2, IPAddress: "127.0.0.2"},
	)

	results, err := c.GetReadings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Err != nil || results[0].Reading == nil {
		t.Errorf("healthy sensor: %+v", results[0])
	}
	if results[1].Err == nil || results[1].Reading != nil {
		t.Errorf("unreachable sensor should report an error: %+v", results[1])
	}
}

func TestGetReadingsListError(t *testing.T) {
	c := collector.New(fakeLister{err: errors.New("db down")}, nil)
	if _, err := c.GetReadings(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
