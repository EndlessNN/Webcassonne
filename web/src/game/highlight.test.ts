import { describe, expect, it } from 'vitest'
import type { GameView, LegalPlacement } from '../api/types'
import { highlightedCells, type Cell } from './highlight'

const IRYNA = 'iryna'
const OLEG = 'oleg'

// legalPlacements, які сервер надсилає для поля з одним стартовим D (обертання 0).
// Ручний розрахунок за docs/tiles.md (той самий, що в game.TestUS11_LegalPlacements).
// U — пряма дорога N–S: лягає праворуч, ліворуч і під D лише як дорога схід–захід.
const U_PLACEMENTS: LegalPlacement[] = [
  { x: -1, y: 0, rotations: [90, 270] },
  { x: 1, y: 0, rotations: [90, 270] },
  { x: 0, y: 1, rotations: [90, 270] },
]
// E — місто на півночі: над D лише 180 (місто до міста), під D — 90, 180, 270.
const E_PLACEMENTS: LegalPlacement[] = [
  { x: 0, y: -1, rotations: [180] },
  { x: 0, y: 1, rotations: [90, 180, 270] },
]

// gameView — хід Ірини, фаза PLACE_TILE, на полі лише стартовий D.
function gameView(legalPlacements: LegalPlacement[], patch: Partial<GameView> = {}): GameView {
  return {
    phase: 'PLACE_TILE',
    turnNumber: 2,
    currentPlayerId: IRYNA,
    deckCount: 69,
    currentTile: { tileTypeId: 'U' },
    discardedTiles: [],
    legalPlacements,
    legalMeepleSegments: [],
    tiles: [{ x: 0, y: 0, tileTypeId: 'D', rotation: 0 }],
    meeples: [],
    finalScores: null,
    ...patch,
  }
}

// Порядок клітинок не входить у контракт — порівнюємо відсортованими за y, потім x.
function sorted(cells: Cell[]): Cell[] {
  return cells.map(({ x, y }) => ({ x, y })).sort((a, b) => a.y - b.y || a.x - b.x)
}

describe('highlightedCells', () => {
  it('підсвітка позицій', () => {
    const cells = highlightedCells(gameView(U_PLACEMENTS), IRYNA, 90)
    expect(sorted(cells)).toEqual([
      { x: -1, y: 0 },
      { x: 1, y: 0 },
      { x: 0, y: 1 },
    ])
  })

  it('жодна клітинка, де сторони не збігаються, не підсвічена', () => {
    // U у початковій орієнтації (0) не лягає ніде; E у 0 — теж.
    expect(highlightedCells(gameView(U_PLACEMENTS), IRYNA, 0)).toEqual([])
    expect(highlightedCells(gameView(E_PLACEMENTS, { currentTile: { tileTypeId: 'E' } }), IRYNA, 0)).toEqual([])
  })

  it('оновлення після обертання', () => {
    // U: у поточній орієнтації (90) підсвічено 3 позиції; після повороту на 180 — жодної.
    expect(highlightedCells(gameView(U_PLACEMENTS), IRYNA, 90)).toHaveLength(3)
    expect(highlightedCells(gameView(U_PLACEMENTS), IRYNA, 180)).toEqual([])

    // E: 180 → над і під D; 270 → лише під D.
    const e = gameView(E_PLACEMENTS, { currentTile: { tileTypeId: 'E' } })
    expect(sorted(highlightedCells(e, IRYNA, 180))).toEqual([
      { x: 0, y: -1 },
      { x: 0, y: 1 },
    ])
    expect(sorted(highlightedCells(e, IRYNA, 270))).toEqual([{ x: 0, y: 1 }])
  })

  it('чужий хід', () => {
    // Зараз хід Ірини — Олег не бачить підсвітки в жодному обертанні.
    for (const r of [0, 90, 180, 270] as const) {
      expect(highlightedCells(gameView(U_PLACEMENTS), OLEG, r)).toEqual([])
    }
  })

  it('фаза PLACE_MEEPLE', () => {
    // api.md: legalPlacements лише у фазі PLACE_TILE; клієнт не підсвічує й тоді, якщо дані ще є.
    expect(highlightedCells(gameView(U_PLACEMENTS, { phase: 'PLACE_MEEPLE' }), IRYNA, 90)).toEqual([])
  })
})
