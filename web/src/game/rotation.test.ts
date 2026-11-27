import { describe, expect, it } from 'vitest'
import type { Rotation } from '../api/types'
import { nextRotation } from './rotation'

// US-10: обертання — локальна дія клієнта, 90° за годинниковою (П-6).
describe('nextRotation', () => {
  it('поворот на 90°', () => {
    expect(nextRotation(0)).toBe(90)
  })

  it('повний оберт', () => {
    // 4 натискання «Повернути» з початкової орієнтації.
    const seen: Rotation[] = []
    let r: Rotation = 0
    for (let i = 0; i < 4; i++) {
      r = nextRotation(r)
      seen.push(r)
    }
    expect(seen).toEqual([90, 180, 270, 0])
  })

  it.each<Rotation>([0, 90, 180, 270])('4 повороти з %i повертають у ту саму орієнтацію', (start) => {
    expect(nextRotation(nextRotation(nextRotation(nextRotation(start))))).toBe(start)
  })
})
