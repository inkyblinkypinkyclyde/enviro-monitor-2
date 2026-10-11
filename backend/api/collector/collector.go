package collector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"enviro-monitor/api/models"
)

const (
	defaultPort    = "80"
	requestTimeout = 5 * time.Second
	maxBodyBytes   = 4 << 10
)

type SensorLister interface {
	Sensors(ctx context.Context) ([]models.Sensor, error)
}

// Result is the outcome of polling one sensor; exactly one of Reading and Err is set.
type Result struct {
	Sensor  models.Sensor
	Reading *models.Reading
	Err     error
}

type Collector struct {
	sensors SensorLister
	client  *http.Client

	// Port the sensors' web servers listen on.
	Port string
}

func New(sensors SensorLister, client *http.Client) *Collector {
	if client == nil {
		client = http.DefaultClient
	}
	return &Collector{sensors: sensors, client: client, Port: defaultPort}
}

// GetReadings polls every sensor concurrently. A failing sensor does not stop the
// others; its error is reported in its Result. The returned error is only for
// failing to list the sensors.
func (c *Collector) GetReadings(ctx context.Context) ([]Result, error) {
	sensors, err := c.sensors.Sensors(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sensors: %w", err)
	}

	results := make([]Result, len(sensors))
	var wg sync.WaitGroup
	for i, sn := range sensors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = Result{Sensor: sn}
			reading, err := c.GetReading(ctx, sn)
			if err != nil {
				results[i].Err = err
				return
			}
			results[i].Reading = &reading
		}()
	}
	wg.Wait()
	return results, nil
}

// statusResponse is the JSON served by the ESP8266 sketch at /status.
type statusResponse struct {
	Hostname    string   `json:"hostname"`
	Temperature *float32 `json:"temperature"`
	Humidity    *float32 `json:"humidity"`
}

// GetReading fetches the current reading from a single sensor.
func (c *Collector) GetReading(ctx context.Context, sensor models.Sensor) (models.Reading, error) {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	url := "http://" + net.JoinHostPort(sensor.IPAddress, c.Port) + "/status"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return models.Reading{}, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return models.Reading{}, fmt.Errorf("sensor %d (%s): %w", sensor.ID, sensor.IPAddress, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.Reading{}, fmt.Errorf("sensor %d (%s): unexpected status %d", sensor.ID, sensor.IPAddress, resp.StatusCode)
	}

	var body statusResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBodyBytes)).Decode(&body); err != nil {
		return models.Reading{}, fmt.Errorf("sensor %d (%s): decode response: %w", sensor.ID, sensor.IPAddress, err)
	}
	if body.Temperature == nil || body.Humidity == nil {
		return models.Reading{}, fmt.Errorf("sensor %d (%s): %w", sensor.ID, sensor.IPAddress, errMissingFields)
	}

	return models.Reading{
		SensorID:      sensor.ID,
		Temperature:   *body.Temperature,
		Humidity:      *body.Humidity,
		CollectedTime: time.Now(),
	}, nil
}

var errMissingFields = errors.New("response missing temperature or humidity")
