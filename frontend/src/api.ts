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
