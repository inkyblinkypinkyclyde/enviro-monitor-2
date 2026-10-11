package models

import (
	"time"

	"github.com/uptrace/bun"
)

type Room struct {
	bun.BaseModel `bun:"table:rooms,alias:r" json:"-"`

	ID       int    `bun:"id,pk,autoincrement" json:"id"`
	RoomName string `bun:"room_name" json:"room_name"`
}

type Sensor struct {
	bun.BaseModel `bun:"table:sensors,alias:s" json:"-"`

	ID     int `bun:"id,pk,autoincrement" json:"id"`
	RoomID int `bun:"room_id" json:"room_id"`
}

type Reading struct {
	bun.BaseModel `bun:"table:dht22_readings,alias:rd" json:"-"`

	ID            int       `bun:"id,pk,autoincrement" json:"id"`
	SensorID      int       `bun:"sensor_id" json:"sensor_id"`
	Temperature   float32   `bun:"temperature" json:"temperature"`
	Humidity      float32   `bun:"humidity" json:"humidity"`
	CollectedTime time.Time `bun:"collected_time" json:"collected_time"`
}

// SensorWithReading is a sensor plus its most recent reading (nil if it has none).
type SensorWithReading struct {
	ID            int      `json:"id"`
	RoomID        int      `json:"room_id"`
	LatestReading *Reading `json:"latest_reading"`
}

// RoomDetail is a room with every sensor in it.
type RoomDetail struct {
	ID       int                 `json:"id"`
	RoomName string              `json:"room_name"`
	Sensors  []SensorWithReading `json:"sensors"`
}
