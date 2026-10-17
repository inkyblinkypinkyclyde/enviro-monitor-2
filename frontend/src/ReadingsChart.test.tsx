import { cleanup, render } from '@testing-library/react'
import { afterEach, describe, expect, it } from 'vitest'
import type { Reading } from './api'
import ReadingsChart from './ReadingsChart'

const NOW = Date.parse('2026-10-17T12:00:00Z')
const HOUR = 60 * 60 * 1000

const reading = (id: number, hoursAgo: number, temperature: number, humidity: number): Reading => ({
  id,
  sensor_id: 1,
  temperature,
  humidity,
  collected_time: new Date(NOW - hoursAgo * HOUR).toISOString(),
})

afterEach(cleanup)

describe('ReadingsChart', () => {
  it('shows an empty message with no readings', () => {
    const { container } = render(<ReadingsChart readings={[]} now={NOW} metric="temperature" />)
    expect(container.textContent).toContain('No readings in the last 24 hours')
    expect(container.querySelector('svg')).toBeNull()
  })

  it('ignores readings older than 24 hours', () => {
    const { container } = render(
      <ReadingsChart readings={[reading(1, 30, 20, 50)]} now={NOW} metric="temperature" />,
    )
    expect(container.querySelector('polyline')).toBeNull()
  })

  it('plots the requested metric only', () => {
    const readings = [reading(1, 2, 10, 90), reading(2, 1, 10, 90)]
    const temp = render(<ReadingsChart readings={readings} now={NOW} metric="temperature" />)
    const tempPoints = temp.container.querySelector('polyline')!.getAttribute('points')
    cleanup()
    const hum = render(<ReadingsChart readings={readings} now={NOW} metric="humidity" />)
    const humPoints = hum.container.querySelector('polyline')!.getAttribute('points')
    expect(tempPoints).not.toEqual(humPoints)
  })

  it('pins out-of-range values to the plot edge', () => {
    const { container } = render(
      <ReadingsChart readings={[reading(1, 1, 100, 50), reading(2, 2, -50, 50)]} now={NOW} metric="temperature" />,
    )
    const ys = container
      .querySelector('polyline')!
      .getAttribute('points')!
      .split(' ')
      .map((p) => Number(p.split(',')[1]))
    expect(Math.min(...ys)).toBe(10)
    expect(Math.max(...ys)).toBe(130)
  })

  it('labels the time axis', () => {
    const { container } = render(
      <ReadingsChart readings={[reading(1, 1, 20, 50)]} now={NOW} metric="humidity" />,
    )
    for (const label of ['-24h', '-18h', '-12h', '-6h', 'now']) {
      expect(container.textContent).toContain(label)
    }
  })

  it('draws gridlines only when zoomed', () => {
    const readings = [reading(1, 1, 20, 50)]
    const plain = render(<ReadingsChart readings={readings} now={NOW} metric="temperature" />)
    expect(plain.container.querySelectorAll('.chart-grid')).toHaveLength(0)
    cleanup()
    const zoomed = render(<ReadingsChart readings={readings} now={NOW} metric="temperature" zoomed />)
    expect(zoomed.container.querySelectorAll('.chart-grid')).toHaveLength(3)
  })
})
