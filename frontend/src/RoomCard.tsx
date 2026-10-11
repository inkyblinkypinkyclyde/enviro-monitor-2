import type { Reading, Room } from './api'

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

export default function RoomCard({ room }: { room: Room }) {
  const reading = latestReading(room)

  return (
    <article className="room-card">
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
    </article>
  )
}
