import { useEffect, useState } from 'react'
import { fetchRooms, type Room } from './api'
import RoomCard from './RoomCard'
import RoomForm from './RoomForm'
import './App.css'

const REFRESH_MS = 30_000

export default function App() {
  const [rooms, setRooms] = useState<Room[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const controller = new AbortController()

    const load = () =>
      fetchRooms(controller.signal)
        .then((data) => {
          setRooms(data)
          setError(null)
        })
        .catch((err: unknown) => {
          if (controller.signal.aborted) return
          setError(err instanceof Error ? err.message : 'Failed to load rooms')
        })

    load()
    const timer = setInterval(load, REFRESH_MS)
    return () => {
      controller.abort()
      clearInterval(timer)
    }
  }, [])

  return (
    <main className="app">
      <h1>Enviro Monitor</h1>
      <RoomForm onCreated={(room) => setRooms((prev) => [...(prev ?? []), room])} />
      {error && <p className="error">{error}</p>}
      {!rooms && !error && <p>Loading…</p>}
      {rooms && rooms.length === 0 && <p>No rooms found.</p>}
      {rooms && rooms.length > 0 && (
        <section className="room-grid">
          {rooms.map((room) => (
            <RoomCard key={room.id} room={room} />
          ))}
        </section>
      )}
    </main>
  )
}
