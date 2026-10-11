package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"enviro-monitor/api/models"
)

type Store interface {
	Ping(ctx context.Context) error
	Rooms(ctx context.Context) ([]models.RoomDetail, error)
	Sensors(ctx context.Context) ([]models.Sensor, error)
	Readings(ctx context.Context, sensorID *int, since *time.Time) ([]models.Reading, error)
}

type Server struct {
	store Store
}

func New(s Store) *Server {
	return &Server{store: s}
}

func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.Health)
	mux.HandleFunc("GET /rooms", s.Rooms)
	mux.HandleFunc("GET /sensors", s.Sensors)
	mux.HandleFunc("GET /readings", s.Readings)
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
