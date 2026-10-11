import { useEffect, useState } from 'react'
import { fetchSensorReadings, type Reading, type Room } from './api'
import ReadingsChart from './ReadingsChart'

const HISTORY_LIMIT = 10
const DAY_MS = 24 * 60 * 60 * 1000

// Newest reading across all sensors in the room.
function latestReading(room: Room): Reading | null {
  let latest: Reading | null = null
  for (const sensor of room.sensors) {
    const r = sensor.latest_reading
    if (r && (!latest || new Date(r.collected_time) > new Date(latest.collected_time))) {
      latest = r
    }
  }
  return latest
}

// Readings from the last 24 hours across the room's sensors, newest first.
function useRoomReadings(room: Room) {
  const [data, setData] = useState<{ readings: Reading[]; now: number } | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()
    const now = Date.now()
    const since = new Date(now - DAY_MS).toISOString()

    Promise.all(room.sensors.map((s) => fetchSensorReadings(s.id, since, controller.signal)))
      .then((lists) => {
        const readings = lists
          .flat()
          .sort((a, b) => b.collected_time.localeCompare(a.collected_time))
        setData({ readings, now })
        setError(null)
      })
      .catch((err: unknown) => {
        if (controller.signal.aborted) return
        setError(err instanceof Error ? err.message : 'Failed to load readings')
      })

    return () => controller.abort()
  }, [room])

  return { data, error }
}

export default function RoomCard({ room }: { room: Room }) {
  const [expanded, setExpanded] = useState(false)
  const reading = latestReading(room)
  const { data, error } = useRoomReadings(room)

  return (
    <article className={`room-card${expanded ? ' expanded' : ''}`}>
      <button
        type="button"
        className="room-toggle"
        aria-expanded={expanded}
        onClick={() => setExpanded((v) => !v)}
      >
        <h2>{room.room_name}</h2>
        {reading ? (
          <>
            <dl className="room-reading">
              <div>
                <dt>Temperature</dt>
                <dd>{reading.temperature.toFixed(1)} °C</dd>
              </div>
              <div>
                <dt>Humidity</dt>
                <dd>{reading.humidity.toFixed(1)} %</dd>
              </div>
            </dl>
            <p className="room-time">
              Updated {new Date(reading.collected_time).toLocaleString()}
            </p>
          </>
        ) : (
          <p className="room-empty">No readings yet</p>
        )}
      </button>

      <div className="room-chart">
        {error && <p className="error">{error}</p>}
        {!data && !error && <p className="chart-empty">Loading…</p>}
        {data && <ReadingsChart readings={data.readings} now={data.now} zoomed={expanded} />}
      </div>

      {expanded && data && (
        <div className="room-history">
          <h3>Recent readings</h3>
          {data.readings.length === 0 && <p>No readings in the last 24 hours</p>}
          {data.readings.length > 0 && (
            <table>
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Temp (°C)</th>
                  <th>Humidity (%)</th>
                </tr>
              </thead>
              <tbody>
                {data.readings.slice(0, HISTORY_LIMIT).map((r) => (
                  <tr key={r.id}>
                    <td>{new Date(r.collected_time).toLocaleString()}</td>
                    <td>{r.temperature.toFixed(1)}</td>
                    <td>{r.humidity.toFixed(1)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}
    </article>
  )
}
