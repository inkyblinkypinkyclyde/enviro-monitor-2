export interface Reading {
  id: number
  sensor_id: number
  temperature: number
  humidity: number
  collected_time: string
}

export interface Sensor {
  id: number
  room_id: number
  ip_address: string
  latest_reading: Reading | null
}

export interface Room {
  id: number
  room_name: string
  sensors: Sensor[]
}

export async function fetchRooms(signal?: AbortSignal): Promise<Room[]> {
  const res = await fetch('/api/rooms', { signal })
  if (!res.ok) {
    throw new Error(`Request failed: ${res.status}`)
  }
  return res.json()
}

export async function createRoom(roomName: string, ipAddress: string): Promise<Room> {
  const res = await fetch('/api/rooms', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ room_name: roomName, ip_address: ipAddress }),
  })
  if (!res.ok) {
    const body = (await res.json().catch(() => null)) as { error?: string } | null
    throw new Error(body?.error ?? `Request failed: ${res.status}`)
  }
  return res.json()
}

export async function fetchSensorReadings(
  sensorId: number,
  since: string,
  signal?: AbortSignal,
): Promise<Reading[]> {
  const params = new URLSearchParams({ sensor_id: String(sensorId), since })
  const res = await fetch(`/api/readings?${params}`, { signal })
  if (!res.ok) {
    throw new Error(`Request failed: ${res.status}`)
  }
  return res.json()
}
