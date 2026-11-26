import type { TileType } from '../api/types'
import { cityShape, hasJunction, roadPath, segmentAnchor, sidesOf } from '../game/tileShapes'

const FIELD = '#8cbf5a'
const CITY = '#c98a4b'
const WALL = '#6b3d12'

// Схематичний тайл 100×100 в обертанні 0 (US-06): поле, дороги, міста з мурами, щити, монастир.
export default function TileShape({ type }: { type: TileType }) {
  return (
    <>
      <rect width={100} height={100} fill={FIELD} />
      {type.segments.map((s, i) =>
        s.type === 'ROAD' ? (
          <g key={i}>
            <path d={roadPath(s)} stroke="#555" strokeWidth={10} fill="none" />
            <path d={roadPath(s)} stroke="#f2f2f2" strokeWidth={7} fill="none" />
          </g>
        ) : null,
      )}
      {hasJunction(type.segments) && <circle cx={50} cy={50} r={7} fill="#777" />}
      {type.segments.map((s, i) => {
        if (s.type !== 'CITY') return null
        const { fill, walls } = cityShape(sidesOf(s))
        return (
          <g key={i}>
            <path d={fill} fill={CITY} />
            {walls.map((w) => (
              <path key={w} d={w} fill="none" stroke={WALL} strokeWidth={2.5} strokeLinecap="round" />
            ))}
          </g>
        )
      })}
      {type.segments.map((s, i) => {
        if (s.type === 'MONASTERY') {
          return (
            <g key={i}>
              <rect x={36} y={36} width={28} height={28} fill="#a33" stroke="#511" strokeWidth={1.5} />
              <path d="M50,40 V60 M43,47 H57" stroke="#fff" strokeWidth={2.5} />
            </g>
          )
        }
        if (s.type === 'CITY' && s.pennant) {
          const [x, y] = segmentAnchor(s)
          return <path key={i} d={`M${x - 5},${y - 6} h10 v7 l-5,5 l-5,-5 Z`} fill="#1f4fa3" stroke="#fff" strokeWidth={1} />
        }
        return null
      })}
    </>
  )
}
