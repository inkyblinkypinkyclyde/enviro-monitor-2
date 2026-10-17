import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as api from './api'
import type { Reading, Room } from './api'
import RoomCard from './RoomCard'

const reading = (id: number, minutesAgo: number, temperature: number, humidity: number): Reading => ({
  id,
  sensor_id: 1,
  temperature,
  humidity,
  collected_time: new Date(Date.now() - minutesAgo * 60_000).toISOString(),
})

const room = (latest: Reading | null): Room => ({
  id: 1,
  room_name: 'Lab',
  sensors: [{ id: 1, room_id: 1, ip_address: '10.0.0.5', latest_reading: latest }],
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('RoomCard', () => {
  it('shows the latest reading', async () => {
    vi.spyOn(api, 'fetchSensorReadings').mockResolvedValue([])
    render(<RoomCard room={room(reading(1, 5, 21.46, 48.2))} />)
    expect(screen.getByText('Lab')).toBeTruthy()
    expect(screen.getByText('21.5 °C')).toBeTruthy()
    expect(screen.getByText('48.2 %')).toBeTruthy()
    await waitFor(() => expect(screen.queryByText('Loading…')).toBeNull())
  })

  it('shows a placeholder when there are no readings', () => {
    vi.spyOn(api, 'fetchSensorReadings').mockResolvedValue([])
    render(<RoomCard room={room(null)} />)
    expect(screen.getByText('No readings yet')).toBeTruthy()
  })

  it('renders separate temperature and humidity charts', async () => {
    vi.spyOn(api, 'fetchSensorReadings').mockResolvedValue([reading(1, 5, 20, 50)])
    render(<RoomCard room={room(reading(1, 5, 20, 50))} />)
    await waitFor(() => expect(screen.getAllByRole('img')).toHaveLength(2))
    expect(screen.getByLabelText(/^Temperature over/)).toBeTruthy()
    expect(screen.getByLabelText(/^Humidity over/)).toBeTruthy()
  })

  it('shows an error when loading readings fails', async () => {
    vi.spyOn(api, 'fetchSensorReadings').mockRejectedValue(new Error('boom'))
    render(<RoomCard room={room(null)} />)
    await waitFor(() => expect(screen.getByText('boom')).toBeTruthy())
  })

  it('expands to show recent readings, newest first', async () => {
    vi.spyOn(api, 'fetchSensorReadings').mockResolvedValue([
      reading(1, 30, 10, 40),
      reading(2, 5, 25, 60),
    ])
    render(<RoomCard room={room(reading(2, 5, 25, 60))} />)
    await waitFor(() => expect(screen.getAllByRole('img')).toHaveLength(2))
    expect(screen.queryByText('Recent readings')).toBeNull()

    fireEvent.click(screen.getByRole('button'))
    expect(screen.getByText('Recent readings')).toBeTruthy()
    const rows = screen.getAllByRole('row').slice(1)
    expect(rows).toHaveLength(2)
    expect(rows[0].textContent).toContain('25.0')
  })
})
