package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Room struct {
	ID       int    `json:"id"`
	RoomName string `json:"room_name"`
}

type Sensor struct {
	ID     int `json:"id"`
	RoomID int `json:"room_id"`
}

type Reading struct {
	ID            int       `json:"id"`
	SensorID      int       `json:"sensor_id"`
	Temperature   float32   `json:"temperature"`
	Humidity      float32   `json:"humidity"`
	CollectedTime time.Time `json:"collected_time"`
}

type server struct {
	db *pgxpool.Pool
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func (s *server) fail(w http.ResponseWriter, err error) {
	log.Printf("query failed: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "database unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) rooms(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT id, room_name FROM rooms ORDER BY id`)
	if err != nil {
		s.fail(w, err)
		return
	}
	defer rows.Close()

	out := []Room{}
	for rows.Next() {
		var x Room
		if err := rows.Scan(&x.ID, &x.RoomName); err != nil {
			s.fail(w, err)
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) sensors(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT id, room_id FROM sensors ORDER BY id`)
	if err != nil {
		s.fail(w, err)
		return
	}
	defer rows.Close()

	out := []Sensor{}
	for rows.Next() {
		var x Sensor
		if err := rows.Scan(&x.ID, &x.RoomID); err != nil {
			s.fail(w, err)
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Optional ?sensor_id= filter; returns the latest 100 readings.
func (s *server) readings(w http.ResponseWriter, r *http.Request) {
	query := `SELECT id, sensor_id, temperature, humidity, collected_time
		FROM dht22_readings
		WHERE ($1::int IS NULL OR sensor_id = $1)
		ORDER BY collected_time DESC
		LIMIT 100`

	var sensorID *int
	if v := r.URL.Query().Get("sensor_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid sensor_id"})
			return
		}
		sensorID = &n
	}

	rows, err := s.db.Query(r.Context(), query, sensorID)
	if err != nil {
		s.fail(w, err)
		return
	}
	defer rows.Close()

	out := []Reading{}
	for rows.Next() {
		var x Reading
		if err := rows.Scan(&x.ID, &x.SensorID, &x.Temperature, &x.Humidity, &x.CollectedTime); err != nil {
			s.fail(w, err)
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func main() {
	dsn := getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/enviro_monitor")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	s := &server{db: pool}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /rooms", s.rooms)
	mux.HandleFunc("GET /sensors", s.sensors)
	mux.HandleFunc("GET /readings", s.readings)

	addr := ":" + getenv("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}
