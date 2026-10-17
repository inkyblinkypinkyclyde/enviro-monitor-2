import type { Reading } from './api'

const WIDTH = 320
const HEIGHT = 150
const PAD = { top: 10, right: 10, bottom: 20, left: 34 }
const DAY_MS = 24 * 60 * 60 * 1000

const METRICS = {
  temperature: { color: '#ef4444', min: 0, max: 40, unit: '°', grid: [10, 20, 30], label: 'Temperature' },
  humidity: { color: '#3b82f6', min: 0, max: 100, unit: '%', grid: [25, 50, 75], label: 'Humidity' },
} as const

export type Metric = keyof typeof METRICS

export default function ReadingsChart({
  readings,
  now,
  metric,
  zoomed = false,
}: {
  readings: Reading[]
  now: number
  metric: Metric
  zoomed?: boolean
}) {
  const { color, min, max, unit, grid, label } = METRICS[metric]

  const start = now - DAY_MS
  const points = readings
    .map((r) => ({ t: new Date(r.collected_time).getTime(), v: r[metric] }))
    .filter((p) => p.t >= start && p.t <= now)
    .sort((a, b) => a.t - b.t)

  if (points.length === 0) {
    return <p className="chart-empty">No readings in the last 24 hours</p>
  }

  const plotW = WIDTH - PAD.left - PAD.right
  const plotH = HEIGHT - PAD.top - PAD.bottom

  const x = (t: number) => PAD.left + ((t - start) / DAY_MS) * plotW
  // Values outside the fixed axis range are pinned to the plot edge.
  const y = (v: number) =>
    PAD.top + (1 - (Math.min(Math.max(v, min), max) - min) / (max - min)) * plotH

  return (
    <svg
      className="readings-chart"
      viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
      role="img"
      aria-label={`${label} over the last 24 hours`}
    >
      <rect x={PAD.left} y={PAD.top} width={plotW} height={plotH} className="chart-frame" />

      {zoomed &&
        grid.map((v) => (
          <g key={v}>
            <line x1={PAD.left} x2={PAD.left + plotW} y1={y(v)} y2={y(v)} className="chart-grid" />
            <text x={PAD.left - 4} y={y(v) + 3} textAnchor="end" fontSize="9" fill={color}>
              {v}
              {unit}
            </text>
          </g>
        ))}

      <polyline
        fill="none"
        stroke={color}
        strokeWidth="1.5"
        points={points.map((p) => `${x(p.t).toFixed(1)},${y(p.v).toFixed(1)}`).join(' ')}
      />

      <g fontSize="9" className="chart-label">
        <text x={PAD.left - 4} y={PAD.top + 3} textAnchor="end" fill={color}>
          {max}
          {unit}
        </text>
        <text x={PAD.left - 4} y={PAD.top + plotH} textAnchor="end" fill={color}>
          {min}
          {unit}
        </text>
        <text x={PAD.left} y={HEIGHT - 6} textAnchor="start">
          -24h
        </text>
        <text x={PAD.left + plotW / 4} y={HEIGHT - 6} textAnchor="middle">
          -18h
        </text>
        <text x={PAD.left + plotW / 2} y={HEIGHT - 6} textAnchor="middle">
          -12h
        </text>
        <text x={PAD.left + (plotW * 3) / 4} y={HEIGHT - 6} textAnchor="middle">
          -6h
        </text>
        <text x={PAD.left + plotW} y={HEIGHT - 6} textAnchor="end">
          now
        </text>
      </g>
    </svg>
  )
}
