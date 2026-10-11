import type { Reading } from './api'

const WIDTH = 320
const HEIGHT = 140
const PAD = { top: 10, right: 34, bottom: 22, left: 34 }
const DAY_MS = 24 * 60 * 60 * 1000

const TEMP_COLOR = '#ef4444'
const HUMIDITY_COLOR = '#3b82f6'

const [T_MIN, T_MAX] = [0, 40]
const [H_MIN, H_MAX] = [30, 100]

const GRID_TEMPS = [10, 20, 30]

// Values outside the fixed axis range are pinned to the plot edge.
const scale = (v: number, min: number, max: number) =>
  1 - (Math.min(Math.max(v, min), max) - min) / (max - min)

export default function ReadingsChart({
  readings,
  now,
  zoomed = false,
}: {
  readings: Reading[]
  now: number
  zoomed?: boolean
}) {
  if (readings.length === 0) {
    return <p className="chart-empty">No readings in the last 24 hours</p>
  }

  const start = now - DAY_MS
  const points = [...readings]
    .map((r) => ({ t: new Date(r.collected_time).getTime(), temp: r.temperature, hum: r.humidity }))
    .filter((p) => p.t >= start && p.t <= now)
    .sort((a, b) => a.t - b.t)

  if (points.length === 0) {
    return <p className="chart-empty">No readings in the last 24 hours</p>
  }

  const plotW = WIDTH - PAD.left - PAD.right
  const plotH = HEIGHT - PAD.top - PAD.bottom

  const x = (t: number) => PAD.left + ((t - start) / DAY_MS) * plotW
  const yTemp = (v: number) => PAD.top + scale(v, T_MIN, T_MAX) * plotH
  const yHum = (v: number) => PAD.top + scale(v, H_MIN, H_MAX) * plotH

  const line = (y: (p: (typeof points)[number]) => number) =>
    points.map((p) => `${x(p.t).toFixed(1)},${y(p).toFixed(1)}`).join(' ')

  return (
    <svg
      className="readings-chart"
      viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
      role="img"
      aria-label="Temperature and humidity over the last 24 hours"
    >
      <rect x={PAD.left} y={PAD.top} width={plotW} height={plotH} className="chart-frame" />

      {zoomed &&
        GRID_TEMPS.map((v) => (
          <g key={v}>
            <line
              x1={PAD.left}
              x2={PAD.left + plotW}
              y1={yTemp(v)}
              y2={yTemp(v)}
              className="chart-grid"
            />
            <text
              x={PAD.left - 4}
              y={yTemp(v) + 3}
              textAnchor="end"
              fontSize="9"
              fill={TEMP_COLOR}
            >
              {v}°
            </text>
          </g>
        ))}

      <polyline fill="none" stroke={TEMP_COLOR} strokeWidth="1.5" points={line((p) => yTemp(p.temp))} />
      <polyline fill="none" stroke={HUMIDITY_COLOR} strokeWidth="1.5" points={line((p) => yHum(p.hum))} />

      <g fontSize="9" className="chart-label">
        <text x={PAD.left - 4} y={PAD.top + 3} textAnchor="end" fill={TEMP_COLOR}>
          {T_MAX}°
        </text>
        <text x={PAD.left - 4} y={PAD.top + plotH} textAnchor="end" fill={TEMP_COLOR}>
          {T_MIN}°
        </text>
        <text x={WIDTH - PAD.right + 4} y={PAD.top + 3} fill={HUMIDITY_COLOR}>
          {H_MAX}%
        </text>
        <text x={WIDTH - PAD.right + 4} y={PAD.top + plotH} fill={HUMIDITY_COLOR}>
          {H_MIN}%
        </text>
        <text x={PAD.left} y={HEIGHT - 6} textAnchor="start">
          -24h
        </text>
        <text x={PAD.left + plotW / 2} y={HEIGHT - 6} textAnchor="middle">
          -12h
        </text>
        <text x={PAD.left + plotW} y={HEIGHT - 6} textAnchor="end">
          now
        </text>
      </g>
    </svg>
  )
}
