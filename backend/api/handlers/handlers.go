package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"enviro-monitor/api/collector"
	"enviro-monitor/api/models"
)

type Store interface {
	Ping(ctx context.Context) error
	Rooms(ctx context.Context) ([]models.RoomDetail, error)
	Sensors(ctx context.Context) ([]models.Sensor, error)
	Readings(ctx context.Context, sensorID *int, since *time.Time) ([]models.Reading, error)
	CreateRoom(ctx context.Context, name, ipAddress string) (models.RoomDetail, error)
	SaveReadings(ctx context.Context, readings []models.Reading) error
}

type Collector interface {
	GetReadings(ctx context.Context) ([]collector.Result, error)
}

type Server struct {
	store     Store
	collector Collector

	// Held while a collection run is in progress so runs never overlap.
	collecting sync.Mutex
}

func New(s Store, c Collector) *Server {
	return &Server{store: s, collector: c}
}

func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.Health)
	mux.HandleFunc("GET /rooms", s.Rooms)
	mux.HandleFunc("POST /rooms", s.CreateRoom)
	mux.HandleFunc("GET /sensors", s.Sensors)
	mux.HandleFunc("GET /readings", s.Readings)
	mux.HandleFunc("POST /collect", s.Collect)
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	log.Printf("query failed: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

func (s *Server) Health(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "database unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) Rooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := s.store.Rooms(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rooms)
}

type createRoomRequest struct {
	RoomName  string `json:"room_name"`
	IPAddress string `json:"ip_address"`
}

const maxRoomNameLen = 100

func (s *Server) CreateRoom(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<10)
	var req createRoomRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	name := strings.TrimSpace(req.RoomName)
	if name == "" || len(name) > maxRoomNameLen {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "room_name is required (max 100 characters)"})
		return
	}
	ip := net.ParseIP(strings.TrimSpace(req.IPAddress))
	if ip == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid ip_address"})
		return
	}

	room, err := s.store.CreateRoom(r.Context(), name, ip.String())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, room)
}

type collectFailure struct {
	SensorID  int    `json:"sensor_id"`
	IPAddress string `json:"ip_address"`
}

type collectResponse struct {
	Collected int              `json:"collected"`
	Failed    []collectFailure `json:"failed"`
}

// Collect polls every sensor and stores the readings it got.
func (s *Server) Collect(w http.ResponseWriter, r *http.Request) {
	if !s.collecting.TryLock() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "collection already in progress"})
		return
	}
	defer s.collecting.Unlock()

	results, err := s.collector.GetReadings(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}

	readings := []models.Reading{}
	failed := []collectFailure{}
	for _, res := range results {
		if res.Err != nil {
			log.Printf("collect: %v", res.Err)
			failed = append(failed, collectFailure{SensorID: res.Sensor.ID, IPAddress: res.Sensor.IPAddress})
			continue
		}
		readings = append(readings, *res.Reading)
	}

	if err := s.store.SaveReadings(r.Context(), readings); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, collectResponse{Collected: len(readings), Failed: failed})
}

func (s *Server) Sensors(w http.ResponseWriter, r *http.Request) {
	sensors, err := s.store.Sensors(r.Context())
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sensors)
}

// Optional ?sensor_id= and ?since= (RFC 3339) filters.
func (s *Server) Readings(w http.ResponseWriter, r *http.Request) {
	var sensorID *int
	if v := r.URL.Query().Get("sensor_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid sensor_id"})
			return
		}
		sensorID = &n
	}

	var since *time.Time
	if v := r.URL.Query().Get("since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid since"})
			return
		}
		since = &t
	}

	readings, err := s.store.Readings(r.Context(), sensorID, since)
	if err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, readings)
}
