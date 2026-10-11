package store

import (
	"context"
	"database/sql"
	"time"

	"enviro-monitor/api/models"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

const readingsLimit = 100

type Store struct {
	db *bun.DB
}

func Open(dsn string) *Store {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	return &Store{db: bun.NewDB(sqldb, pgdialect.New())}
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Rooms returns every room with its sensors and each sensor's most recent reading.
func (s *Store) Rooms(ctx context.Context) ([]models.RoomDetail, error) {
	rooms := []models.Room{}
	if err := s.db.NewSelect().Model(&rooms).Order("id ASC").Scan(ctx); err != nil {
		return nil, err
	}

	sensors, err := s.Sensors(ctx)
	if err != nil {
		return nil, err
	}

	latest := []models.Reading{}
	err = s.db.NewSelect().
		Model(&latest).
		DistinctOn("sensor_id").
		Order("sensor_id ASC", "collected_time DESC").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	latestBySensor := make(map[int]*models.Reading, len(latest))
	for i := range latest {
		latestBySensor[latest[i].SensorID] = &latest[i]
	}

	out := make([]models.RoomDetail, len(rooms))
	idx := make(map[int]int, len(rooms))
	for i, r := range rooms {
		out[i] = models.RoomDetail{ID: r.ID, RoomName: r.RoomName, Sensors: []models.SensorWithReading{}}
		idx[r.ID] = i
	}
	for _, sn := range sensors {
		i, ok := idx[sn.RoomID]
		if !ok {
			continue
		}
		out[i].Sensors = append(out[i].Sensors, models.SensorWithReading{
			ID:            sn.ID,
			RoomID:        sn.RoomID,
			LatestReading: latestBySensor[sn.ID],
		})
	}
	return out, nil
}

func (s *Store) Sensors(ctx context.Context) ([]models.Sensor, error) {
	sensors := []models.Sensor{}
	err := s.db.NewSelect().Model(&sensors).Order("id ASC").Scan(ctx)
	return sensors, err
}

// Readings returns readings newest first, filtered by sensorID when non-nil.
// With since set, every reading at or after it is returned; otherwise only the latest readingsLimit.
func (s *Store) Readings(ctx context.Context, sensorID *int, since *time.Time) ([]models.Reading, error) {
	readings := []models.Reading{}
	q := s.db.NewSelect().
		Model(&readings).
		Order("collected_time DESC")
	if sensorID != nil {
		q = q.Where("sensor_id = ?", *sensorID)
	}
	if since != nil {
		q = q.Where("collected_time >= ?", *since)
	} else {
		q = q.Limit(readingsLimit)
	}
	err := q.Scan(ctx)
	return readings, err
}
