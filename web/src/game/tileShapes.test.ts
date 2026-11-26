import { describe, expect, it } from 'vitest'
import type { FeatureType, Segment } from '../api/types'
import { cityShape, edgePoint, hasJunction, sidesOf, type Side } from './tileShapes'

// US-06: схематичний тайл із сегментів каталогу. Перевіряємо лише те, що задають docs/data-model.md і docs/tiles.md
// (тайл 100×100, північ угорі, точки країв за годинниковою, стикування дзеркальних точок, мури — межа міста з полем),
// а не конкретні SVG-шляхи: їхній вигляд перевіряється вручну.

function seg(type: FeatureType, points: string[]): Segment {
  return { type, points, pennant: false, adjacentCities: [] }
}

describe('edgePoint', () => {
  it('центральні точки сторін — середини країв тайла', () => {
    expect(edgePoint('N1')).toEqual([50, 0])
    expect(edgePoint('E1')).toEqual([100, 50])
    expect(edgePoint('S1')).toEqual([50, 100])
    expect(edgePoint('W1')).toEqual([0, 50])
  })

  it('точки йдуть за годинниковою стрілкою (tiles.md, «Умовні позначення»)', () => {
    const [n0, n1, n2] = ['N0', 'N1', 'N2'].map(edgePoint)
    const [e0, e1, e2] = ['E0', 'E1', 'E2'].map(edgePoint)
    const [s0, s1, s2] = ['S0', 'S1', 'S2'].map(edgePoint)
    const [w0, w1, w2] = ['W0', 'W1', 'W2'].map(edgePoint)
    // верх зліва направо, правий край згори вниз, низ справа наліво, лівий край знизу вгору
    expect([n0[1], n1[1], n2[1]]).toEqual([0, 0, 0])
    expect(n0[0] < n1[0] && n1[0] < n2[0]).toBe(true)
    expect([e0[0], e1[0], e2[0]]).toEqual([100, 100, 100])
    expect(e0[1] < e1[1] && e1[1] < e2[1]).toBe(true)
    expect([s0[1], s1[1], s2[1]]).toEqual([100, 100, 100])
    expect(s0[0] > s1[0] && s1[0] > s2[0]).toBe(true)
    expect([w0[0], w1[0], w2[0]]).toEqual([0, 0, 0])
    expect(w0[1] > w1[1] && w1[1] > w2[1]).toBe(true)
  })

  it('дзеркальні точки стикування лежать навпроти (N0↔S2, N2↔S0, E0↔W2, E2↔W0)', () => {
    expect(edgePoint('N0')[0]).toBeCloseTo(edgePoint('S2')[0])
    expect(edgePoint('N2')[0]).toBeCloseTo(edgePoint('S0')[0])
    expect(edgePoint('E0')[1]).toBeCloseTo(edgePoint('W2')[1])
    expect(edgePoint('E2')[1]).toBeCloseTo(edgePoint('W0')[1])
  })
})

describe('sidesOf', () => {
  it.each<[string, Segment, Side[]]>([
    ['U, дорога N1–S1', seg('ROAD', ['N1', 'S1']), ['N', 'S']],
    ['F, місто зі сходу на захід', seg('CITY', ['E0', 'E1', 'E2', 'W0', 'W1', 'W2']), ['E', 'W']],
    ['W, поле на півночі', seg('FIELD', ['W2', 'N0', 'N1', 'N2', 'E0']), ['N', 'E', 'W']],
    ['B, монастир без точок', seg('MONASTERY', []), []],
  ])('%s', (_, s, want) => {
    // порядок сторін контракт не задає — порівнюємо як множину
    expect([...sidesOf(s)].sort()).toEqual([...want].sort())
  })
})

describe('cityShape', () => {
  // Мур — межа міста з полем: стільки мурів, скільки суцільних ділянок не-міських сторін між міськими.
  it.each<[string, Side[], number]>([
    ['C, місто на всіх сторонах — мурів немає', ['N', 'E', 'S', 'W'], 0],
    ['E, місто на півночі', ['N'], 1],
    ['N, кутове місто північ–захід', ['N', 'W'], 1],
    ['R, місто на півночі, сході й заході', ['N', 'E', 'W'], 1],
    ['F, місто через центр зі сходу на захід — поля на півночі й півдні', ['E', 'W'], 2],
  ])('%s', (_, sides, walls) => {
    const shape = cityShape(sides)
    expect(shape.fill).not.toBe('')
    expect(shape.walls).toHaveLength(walls)
  })
})

describe('hasJunction', () => {
  // tiles.md: дорога з однією точкою закінчується на перехресті, біля монастиря або біля міста;
  // перехрестя — коли так закінчуються три чи чотири дороги (L, W, X).
  const field = seg('FIELD', ['N0'])
  it.each<[string, Segment[], boolean]>([
    ['X, чотири дороги', [seg('ROAD', ['N1']), seg('ROAD', ['E1']), seg('ROAD', ['S1']), seg('ROAD', ['W1']), field], true],
    ['W, три дороги', [seg('ROAD', ['E1']), seg('ROAD', ['S1']), seg('ROAD', ['W1']), field], true],
    ['L, місто і три дороги', [seg('CITY', ['N0', 'N1', 'N2']), seg('ROAD', ['E1']), seg('ROAD', ['S1']), seg('ROAD', ['W1'])], true],
    ['U, пряма дорога', [seg('ROAD', ['N1', 'S1']), field], false],
    ['A, дорога від монастиря', [seg('MONASTERY', []), seg('ROAD', ['S1']), field], false],
    ['S, дорога від міста', [seg('CITY', ['N0', 'N1', 'N2', 'E0', 'E1', 'E2', 'W0', 'W1', 'W2']), seg('ROAD', ['S1'])], false],
  ])('%s', (_, segments, want) => {
    expect(hasJunction(segments)).toBe(want)
  })
})
