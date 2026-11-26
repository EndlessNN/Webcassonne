import type { Segment } from '../api/types'

// Геометрія схематичного тайла 100×100 (обертання 0, північ угорі) із сегментів каталогу.
// Точки країв — docs/data-model.md: N0 N1 N2 зліва направо вгорі, далі за годинниковою стрілкою.

export type Point = readonly [number, number]
const SIDES = ['N', 'E', 'S', 'W'] as const
export type Side = (typeof SIDES)[number]

const CENTER: Point = [50, 50]
const MID: Record<Side, Point> = { N: [50, 0], E: [100, 50], S: [50, 100], W: [0, 50] }
// Кути сторони за годинниковою стрілкою: [початок, кінець].
const CORNERS: Record<Side, readonly [Point, Point]> = {
  N: [[0, 0], [100, 0]],
  E: [[100, 0], [100, 100]],
  S: [[100, 100], [0, 100]],
  W: [[0, 100], [0, 0]],
}

const lerp = (a: Point, b: Point, t: number): Point => [a[0] + (b[0] - a[0]) * t, a[1] + (b[1] - a[1]) * t]
const fmt = (p: Point) => `${p[0]},${p[1]}`
const sideOf = (p: string): Side => SIDES.find((s) => s === p[0]) ?? 'N'

export function edgePoint(p: string): Point {
  const [a, b] = CORNERS[sideOf(p)]
  return lerp(a, b, [1 / 6, 1 / 2, 5 / 6][Number(p[1])] ?? 1 / 2)
}

export function sidesOf(seg: Segment): Side[] {
  return SIDES.filter((s) => seg.points.some((p) => p[0] === s))
}

export interface CityShape {
  fill: string // шлях заливки міста
  walls: string[] // мури: лише межа міста з полем чи дорогою; уздовж краю тайла мурів немає
}

// Місто на кількох сторонах — одна фігура; між сусідніми міськими сторонами межі немає,
// а між міськими сторонами через неміські проходить мур-крива.
export function cityShape(sides: Side[]): CityShape {
  if (sides.length === 4) return { fill: 'M0,0 H100 V100 H0 Z', walls: [] }
  const inCity = (s: Side) => sides.includes(s)
  let i = SIDES.findIndex((s, j) => inCity(s) && !inCity(SIDES[(j + 3) % 4]))
  if (i < 0) return { fill: '', walls: [] }
  let d = `M${fmt(CORNERS[SIDES[i]][0])}`
  const walls: string[] = []
  for (let steps = 0; steps < 4; ) {
    const s = SIDES[i]
    d += ` L${fmt(CORNERS[s][1])}`
    let k = 0
    while (!inCity(SIDES[(i + 1 + k) % 4])) k++
    if (k > 0) {
      const from = CORNERS[s][1]
      const to = CORNERS[SIDES[(i + 1 + k) % 4]][0]
      const ctrl =
        k === 1
          ? lerp(MID[SIDES[(i + 1) % 4]], CENTER, 1.3) // пропущено одну сторону: поле-клин
          : k === 2
            ? CENTER // кутове місто: межа по діагоналі
            : lerp(MID[s], CENTER, 0.7) // місто на одній стороні
      d += ` Q${fmt(ctrl)} ${fmt(to)}`
      walls.push(`M${fmt(from)} Q${fmt(ctrl)} ${fmt(to)}`)
    }
    steps += 1 + k
    i = (i + 1 + k) % 4
  }
  return { fill: `${d} Z`, walls }
}

export function roadPath(seg: Segment): string {
  const pts = seg.points.map(edgePoint)
  return pts.length === 2 ? `M${fmt(pts[0])} Q${fmt(CENTER)} ${fmt(pts[1])}` : `M${fmt(pts[0])} L${fmt(CENTER)}`
}

// Перехрестя: три й більше доріг закінчуються в центрі тайла.
export function hasJunction(segments: Segment[]): boolean {
  return segments.filter((s) => s.type === 'ROAD' && s.points.length === 1).length >= 3
}

// Точка всередині сегмента — для міпла і щита.
export function segmentAnchor(seg: Segment): Point {
  if (seg.type === 'MONASTERY') return CENTER
  if (seg.type === 'CITY') {
    const sides = sidesOf(seg)
    const mids = sides.map((s) => MID[s])
    const c: Point = [mids.reduce((a, p) => a + p[0], 0) / mids.length, mids.reduce((a, p) => a + p[1], 0) / mids.length]
    return sides.length === 1 ? lerp(c, CENTER, 0.17) : c
  }
  const pts = seg.points.map(edgePoint)
  if (seg.type === 'ROAD') {
    return pts.length === 2 ? lerp(lerp(pts[0], CENTER, 0.25), lerp(CENTER, pts[1], 0.25), 0.25) : lerp(pts[0], CENTER, 0.3)
  }
  // Поле: середня точка найдовшого відрізка суміжних точок краю, на 10 одиниць углиб тайла.
  const all = SIDES.flatMap((s) => [0, 1, 2].map((k) => `${s}${k}`))
  const has = all.map((p) => seg.points.includes(p))
  const start = has.every(Boolean) ? 0 : has.findIndex((h, j) => h && !has[(j + 11) % 12])
  let best: string[] = []
  let run: string[] = []
  for (let n = 0; n < 12; n++) {
    const j = (start + n) % 12
    run = has[j] ? [...run, all[j]] : []
    if (run.length > best.length) best = run
  }
  const p = best[Math.floor(best.length / 2)] ?? 'N1'
  const [x, y] = edgePoint(p)
  const inward: Record<Side, Point> = { N: [x, y + 10], E: [x - 10, y], S: [x, y - 10], W: [x + 10, y] }
  return inward[sideOf(p)]
}
