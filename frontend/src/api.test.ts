import { afterEach, describe, expect, it, vi } from 'vitest'
import { createRoom, fetchRooms, fetchSensorReadings } from './api'

function mockFetch(body: unknown, init: { ok?: boolean; status?: number } = {}) {
  const fn = vi.fn().mockResolvedValue({
    ok: init.ok ?? true,
    status: init.status ?? 200,
    json: () => Promise.resolve(body),
  })
  vi.stubGlobal('fetch', fn)
  return fn
}

afterEach(() => vi.unstubAllGlobals())

describe('fetchRooms', () => {
  it('returns parsed rooms', async () => {
    const fn = mockFetch([{ id: 1, room_name: 'Lab', sensors: [] }])
    await expect(fetchRooms()).resolves.toEqual([{ id: 1, room_name: 'Lab', sensors: [] }])
    expect(fn).toHaveBeenCalledWith('/api/rooms', { signal: undefined })
  })

  it('throws on a non-ok response', async () => {
    mockFetch(null, { ok: false, status: 500 })
    await expect(fetchRooms()).rejects.toThrow('Request failed: 500')
  })
})

describe('createRoom', () => {
  it('posts the room name and ip address', async () => {
    const fn = mockFetch({ id: 2, room_name: 'Kitchen', sensors: [] })
    await createRoom('Kitchen', '10.0.0.5')
    const [url, opts] = fn.mock.calls[0]
    expect(url).toBe('/api/rooms')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body)).toEqual({ room_name: 'Kitchen', ip_address: '10.0.0.5' })
  })

  it('surfaces the server error message', async () => {
    mockFetch({ error: 'ip already in use' }, { ok: false, status: 409 })
    await expect(createRoom('Kitchen', '10.0.0.5')).rejects.toThrow('ip already in use')
  })

  it('falls back to the status when the body has no error', async () => {
    mockFetch(null, { ok: false, status: 400 })
    await expect(createRoom('Kitchen', 'x')).rejects.toThrow('Request failed: 400')
  })
})

describe('fetchSensorReadings', () => {
  it('encodes sensor_id and since in the query string', async () => {
    const fn = mockFetch([])
    await fetchSensorReadings(7, '2026-10-16T00:00:00.000Z')
    const url = new URL(fn.mock.calls[0][0], 'http://localhost')
    expect(url.pathname).toBe('/api/readings')
    expect(url.searchParams.get('sensor_id')).toBe('7')
    expect(url.searchParams.get('since')).toBe('2026-10-16T00:00:00.000Z')
  })
})
