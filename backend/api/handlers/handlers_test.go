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

	"enviro-monitor/api/collector"
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

	gotName, gotIP string

	saved    []models.Reading
	saveErr  error
	saveCall int
}

func (f *fakeStore) SaveReadings(_ context.Context, r []models.Reading) error {
	f.saveCall++
	f.saved = r
	return f.saveErr
}

type fakeCollector struct {
	results []collector.Result
	err     error
	block   chan struct{}
	entered chan struct{}
}

func (f *fakeCollector) GetReadings(context.Context) ([]collector.Result, error) {
	if f.block != nil {
		close(f.entered)
		<-f.block
	}
	return f.results, f.err
}

func postCollect(st handlers.Store, c handlers.Collector) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handlers.New(st, c).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/collect", nil))
	return rec
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

func (f *fakeStore) CreateRoom(_ context.Context, name, ip string) (models.RoomDetail, error) {
	f.gotName, f.gotIP = name, ip
	return models.RoomDetail{
		ID: 7, RoomName: name,
		Sensors: []models.SensorWithReading{{ID: 8, RoomID: 7, IPAddress: ip}},
	}, f.err
}

func post(t *testing.T, st handlers.Store, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(body))
	handlers.New(st, nil).Routes().ServeHTTP(rec, req)
	return rec
}

func do(t *testing.T, st handlers.Store, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handlers.New(st, nil).Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
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
	st := &fakeStore{sensors: []models.Sensor{{ID: 2, RoomID: 1, IPAddress: "192.168.1.101"}}}
	rec := do(t, st, "/sensors")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var got []models.Sensor
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != 2 || got[0].RoomID != 1 || got[0].IPAddress != "192.168.1.101" {
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

func TestCreateRoom(t *testing.T) {
	st := &fakeStore{}
	rec := post(t, st, `{"room_name":"  Garage ","ip_address":"192.168.1.50"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body)
	}
	if st.gotName != "Garage" || st.gotIP != "192.168.1.50" {
		t.Errorf("store got %q, %q", st.gotName, st.gotIP)
	}

	var got models.RoomDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != 7 || len(got.Sensors) != 1 || got.Sensors[0].IPAddress != "192.168.1.50" {
		t.Errorf("got %+v", got)
	}
}

func TestCreateRoomValidation(t *testing.T) {
	bodies := []string{
		`not json`,
		`{"room_name":"","ip_address":"192.168.1.50"}`,
		`{"room_name":"   ","ip_address":"192.168.1.50"}`,
		`{"room_name":"Garage","ip_address":""}`,
		`{"room_name":"Garage","ip_address":"999.1.1.1"}`,
		`{"room_name":"Garage","ip_address":"not-an-ip"}`,
		`{"room_name":"` + strings.Repeat("a", 101) + `","ip_address":"192.168.1.50"}`,
	}
	for _, b := range bodies {
		st := &fakeStore{}
		if rec := post(t, st, b); rec.Code != http.StatusBadRequest {
			t.Errorf("%.40s: status = %d, want 400", b, rec.Code)
		}
		if st.gotName != "" {
			t.Errorf("%.40s: store was called", b)
		}
	}
}

func TestCreateRoomStoreError(t *testing.T) {
	rec := post(t, &fakeStore{err: errors.New("boom")}, `{"room_name":"Garage","ip_address":"192.168.1.50"}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("body leaks error: %s", rec.Body)
	}
}

func TestCollect(t *testing.T) {
	good := &models.Reading{SensorID: 1, Temperature: 21, Humidity: 50}
	c := &fakeCollector{results: []collector.Result{
		{Sensor: models.Sensor{ID: 1, IPAddress: "10.0.0.1"}, Reading: good},
		{Sensor: models.Sensor{ID: 2, IPAddress: "10.0.0.2"}, Err: errors.New("dial tcp 10.0.0.2:80: timeout")},
	}}
	st := &fakeStore{}

	rec := postCollect(st, c)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rec.Code, rec.Body)
	}
	if len(st.saved) != 1 || st.saved[0].SensorID != 1 {
		t.Errorf("saved %+v", st.saved)
	}

	var got struct {
		Collected int `json:"collected"`
		Failed    []struct {
			SensorID  int    `json:"sensor_id"`
			IPAddress string `json:"ip_address"`
		} `json:"failed"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Collected != 1 || len(got.Failed) != 1 || got.Failed[0].SensorID != 2 {
		t.Errorf("got %+v", got)
	}
	if strings.Contains(rec.Body.String(), "timeout") {
		t.Errorf("body leaks sensor error: %s", rec.Body)
	}
}

func TestCollectNoSensors(t *testing.T) {
	rec := postCollect(&fakeStore{}, &fakeCollector{})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"failed":[]`) {
		t.Errorf("failed should be an empty array: %s", rec.Body)
	}
}

func TestCollectErrors(t *testing.T) {
	if rec := postCollect(&fakeStore{}, &fakeCollector{err: errors.New("boom")}); rec.Code != http.StatusInternalServerError {
		t.Errorf("collector error: status = %d, want 500", rec.Code)
	}

	c := &fakeCollector{results: []collector.Result{{Sensor: models.Sensor{ID: 1}, Reading: &models.Reading{SensorID: 1}}}}
	if rec := postCollect(&fakeStore{saveErr: errors.New("boom")}, c); rec.Code != http.StatusInternalServerError {
		t.Errorf("save error: status = %d, want 500", rec.Code)
	}
}

func TestCollectRejectsOverlap(t *testing.T) {
	c := &fakeCollector{block: make(chan struct{}), entered: make(chan struct{})}
	srv := handlers.New(&fakeStore{}, c)

	done := make(chan int)
	go func() {
		rec := httptest.NewRecorder()
		srv.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/collect", nil))
		done <- rec.Code
	}()
	<-c.entered

	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/collect", nil))
	if rec.Code != http.StatusConflict {
		t.Errorf("overlapping run: status = %d, want 409", rec.Code)
	}

	close(c.block)
	if code := <-done; code != http.StatusOK {
		t.Errorf("first run: status = %d, want 200", code)
	}
}

func TestUnknownRouteAndMethod(t *testing.T) {
	if rec := do(t, &fakeStore{}, "/nope"); rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/rooms", nil)
	handlers.New(&fakeStore{}, nil).Routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}
