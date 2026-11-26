import { useRef, useState, type PointerEvent } from 'react'
import type { MeepleView, PlacedTileView, Rotation, TileType } from '../api/types'
import type { Cell } from '../game/highlight'
import { segmentAnchor } from '../game/tileShapes'
import TileShape from './TileShape'

const TILE = 100 // розмір тайла в одиницях SVG
const VIEW_W = 1000
const VIEW_H = 640
const DRAG_THRESHOLD = 4 // px: менший рух — це клік, а не перетягування

interface Props {
  tiles: PlacedTileView[]
  meeples: MeepleView[]
  catalog: Map<string, TileType>
  meepleColor: (playerId: string) => string
  highlighted: Cell[]
  ghost: { type: TileType; rotation: Rotation } | null // поточний тайл під курсором на підсвіченій клітинці
  onCellClick: ((cell: Cell) => void) | null // null — зараз не можна класти тайл
}

interface Drag {
  pointerId: number
  startX: number
  startY: number
  center: { x: number; y: number }
  moved: boolean
}

// Поле на SVG (ADR-0002): тайли й міпли (US-06), панорамування мишею зміною viewBox (US-08),
// підсвітка з legalPlacements (US-11), клік по клітинці (US-12). Стикування не перевіряє — це робить сервер.
export default function Board({ tiles, meeples, catalog, meepleColor, highlighted, ghost, onCellClick }: Props) {
  const svgRef = useRef<SVGSVGElement>(null)
  const [center, setCenter] = useState({ x: TILE / 2, y: TILE / 2 }) // центр стартового тайла
  const [drag, setDrag] = useState<Drag | null>(null)
  const [hover, setHover] = useState<Cell | null>(null)

  const viewX = center.x - VIEW_W / 2
  const viewY = center.y - VIEW_H / 2

  function scale(): number {
    const width = svgRef.current?.getBoundingClientRect().width ?? VIEW_W
    return VIEW_W / width
  }

  function cellAt(e: PointerEvent<SVGSVGElement>): Cell | null {
    const rect = svgRef.current?.getBoundingClientRect()
    if (!rect) return null
    const k = scale()
    return {
      x: Math.floor((viewX + (e.clientX - rect.left) * k) / TILE),
      y: Math.floor((viewY + (e.clientY - rect.top) * k) / TILE),
    }
  }

  function onPointerDown(e: PointerEvent<SVGSVGElement>) {
    if (e.button !== 0) return
    e.currentTarget.setPointerCapture(e.pointerId)
    setDrag({ pointerId: e.pointerId, startX: e.clientX, startY: e.clientY, center, moved: false })
  }

  function onPointerMove(e: PointerEvent<SVGSVGElement>) {
    if (!drag || drag.pointerId !== e.pointerId) {
      setHover(cellAt(e))
      return
    }
    const dx = e.clientX - drag.startX
    const dy = e.clientY - drag.startY
    const moved = drag.moved || Math.abs(dx) + Math.abs(dy) > DRAG_THRESHOLD
    if (moved !== drag.moved) setDrag({ ...drag, moved })
    if (moved) {
      const k = scale()
      // Поле рухається разом з мишею: видима область зміщується проти руху курсора.
      setCenter({ x: drag.center.x - dx * k, y: drag.center.y - dy * k })
    }
  }

  function onPointerUp(e: PointerEvent<SVGSVGElement>) {
    if (!drag || drag.pointerId !== e.pointerId) return
    setDrag(null)
    if (!drag.moved && onCellClick) {
      const cell = cellAt(e)
      if (cell) onCellClick(cell)
    }
  }

  const isHighlighted = (c: Cell | null) => c !== null && highlighted.some((h) => h.x === c.x && h.y === c.y)

  return (
    <svg
      ref={svgRef}
      className="board"
      viewBox={`${viewX} ${viewY} ${VIEW_W} ${VIEW_H}`}
      style={{ cursor: drag?.moved ? 'grabbing' : 'grab' }}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onPointerCancel={() => setDrag(null)}
      onPointerLeave={() => setHover(null)}
    >
      <rect x={viewX} y={viewY} width={VIEW_W} height={VIEW_H} fill="#e9e4d8" />
      {tiles.map((t) => {
        const type = catalog.get(t.tileTypeId)
        if (!type) return null
        return (
          <g key={`${t.x},${t.y}`} transform={`translate(${t.x * TILE} ${t.y * TILE}) rotate(${t.rotation} 50 50)`}>
            <TileShape type={type} />
            {meeples
              .filter((m) => m.x === t.x && m.y === t.y)
              .map((m) => {
                const seg = type.segments[m.segment]
                if (!seg) return null
                const [x, y] = segmentAnchor(seg)
                return <circle key={m.segment} cx={x} cy={y} r={9} fill={meepleColor(m.playerId)} stroke="#fff" strokeWidth={2} />
              })}
          </g>
        )
      })}
      {highlighted.map((c) => (
        <rect
          key={`h${c.x},${c.y}`}
          x={c.x * TILE + 3}
          y={c.y * TILE + 3}
          width={TILE - 6}
          height={TILE - 6}
          fill="rgba(255, 235, 59, 0.35)"
          stroke="#f9a825"
          strokeWidth={4}
        />
      ))}
      {ghost && hover && isHighlighted(hover) && (
        <g
          transform={`translate(${hover.x * TILE} ${hover.y * TILE}) rotate(${ghost.rotation} 50 50)`}
          opacity={0.7}
          pointerEvents="none"
        >
          <TileShape type={ghost.type} />
        </g>
      )}
    </svg>
  )
}
