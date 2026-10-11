package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"enviro-monitor/api/collector"
	"enviro-monitor/api/handlers"
	"enviro-monitor/api/store"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	dsn := getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/enviro_monitor?sslmode=disable")

	st := store.Open(dsn)
	defer st.Close()

	addr := ":" + getenv("PORT", "8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           handlers.New(st, collector.New(st, nil)).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}
