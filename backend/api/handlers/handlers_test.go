package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"enviro-monitor/api/handlers"
	"enviro-monitor/api/models"
)

type fakeStore struct {
	pingErr error
	err     error
	rooms   []models.RoomDetail
	sensors []models.Sensor
	reads   []models.Reading

	gotSensorID *int
	gotSince    *time.Time
}

func (f *fakeStore) Ping(context.Context) error { return f.pingErr }

func (f *fakeStore) Rooms(context.Context) ([]models.RoomDetail, error) {
	return f.rooms, f.err
}

func (f *fakeStore) Sensors(context.Context) ([]models.Sensor, error) {
	return f.sensors, f.err
}

func (f *fakeStore) Readings(_ context.Context, sensorID *int, since *time.Time) ([]models.Reading, error) {
	f.gotSensorID = sensorID
	f.gotSince = since
	return f.reads, f.err
}

func do(t *testing.T, st handlers.Store, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handlers.New(st).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestHealth(t *testing.T) {
	rec := do(t, &fakeStore{}, "/health")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	rec = do(t, &fakeStore{pingErr: errors.New("down")}, "/health")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestRooms(t *testing.T) {
	st := &fakeStore{rooms: []models.RoomDetail{{
		ID: 1, RoomName: "Kitchen",
		Sensors: []models.SensorWithReading{
			{ID: 4, RoomID: 1, LatestReading: &models.Reading{ID: 9, SensorID: 4, Temperature: 22.5}},
			{ID: 5, RoomID: 1},
		},
	}}}
	rec := do(t, st, "/rooms")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q", ct)
	}

	var got []models.RoomDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].RoomName != "Kitchen" || len(got[0].Sensors) != 2 {
		t.Fatalf("got %+v", got)
	}
	if lr := got[0].Sensors[0].LatestReading; lr == nil || lr.Temperature != 22.5 {
		t.Errorf("latest reading = %+v", lr)
	}
	if got[0].Sensors[1].LatestReading != nil {
		t.Errorf("expected null latest_reading for sensor without readings")
	}
}

func TestSensors(t *testing.T) {
	st := &fakeStore{sensors: []models.Sensor{{ID: 2, RoomID: 1}}}
	rec := do(t, st, "/sensors")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var got []models.Sensor
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != 2 || got[0].RoomID != 1 {
		t.Errorf("got %+v", got)
	}
}

func TestReadings(t *testing.T) {
	st := &fakeStore{reads: []models.Reading{{
		ID: 1, SensorID: 3, Temperature: 21.5, Humidity: 50, CollectedTime: time.Now(),
	}}}

	rec := do(t, st, "/readings")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if st.gotSensorID != nil {
		t.Errorf("sensorID = %d, want nil", *st.gotSensorID)
	}

	rec = do(t, st, "/readings?sensor_id=3")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if st.gotSensorID == nil || *st.gotSensorID != 3 {
		t.Errorf("sensorID = %v, want 3", st.gotSensorID)
	}
}

func TestReadingsSince(t *testing.T) {
	st := &fakeStore{}

	rec := do(t, st, "/readings?since=2026-10-10T12:00:00Z")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	want := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	if st.gotSince == nil || !st.gotSince.Equal(want) {
		t.Errorf("since = %v, want %v", st.gotSince, want)
	}

	if rec := do(t, st, "/readings?since=yesterday"); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestReadingsInvalidSensorID(t *testing.T) {
	rec := do(t, &fakeStore{}, "/readings?sensor_id=abc")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestStoreErrorReturns500(t *testing.T) {
	for _, path := range []string{"/rooms", "/sensors", "/readings"} {
		rec := do(t, &fakeStore{err: errors.New("boom")}, path)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("%s: status = %d, want 500", path, rec.Code)
		}
		if body := rec.Body.String(); !json.Valid([]byte(body)) || strings.Contains(body, "boom") {
			t.Errorf("%s: body leaks or is invalid: %q", path, body)
		}
	}
}

func TestUnknownRouteAndMethod(t *testing.T) {
	if rec := do(t, &fakeStore{}, "/nope"); rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms", nil)
	handlers.New(&fakeStore{}).Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
