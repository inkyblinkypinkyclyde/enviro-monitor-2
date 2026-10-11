import { useState, type FormEvent } from 'react'
import { createRoom, type Room } from './api'

export default function RoomForm({ onCreated }: { onCreated: (room: Room) => void }) {
  const [roomName, setRoomName] = useState('')
  const [ipAddress, setIpAddress] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setError(null)
    try {
      const room = await createRoom(roomName.trim(), ipAddress.trim())
      onCreated(room)
      setRoomName('')
      setIpAddress('')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add room')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <form className="room-form" onSubmit={handleSubmit}>
      <label>
        Room name
        <input
          type="text"
          value={roomName}
          onChange={(e) => setRoomName(e.target.value)}
          maxLength={100}
          required
        />
      </label>
      <label>
        Sensor IP address
        <input
          type="text"
          value={ipAddress}
          onChange={(e) => setIpAddress(e.target.value)}
          placeholder="192.168.1.50"
          required
        />
      </label>
      <button type="submit" disabled={submitting}>
        {submitting ? 'Adding…' : 'Add room'}
      </button>
      {error && <p className="error">{error}</p>}
    </form>
  )
}
