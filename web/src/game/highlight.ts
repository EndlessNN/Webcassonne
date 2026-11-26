import type { GameView, Rotation } from '../api/types'

export interface Cell {
  x: number
  y: number
}

// US-11: які клітинки підсвітити. Позиції бере лише з legalPlacements сервера (клієнт стикування не перевіряє):
// лише в поточному обертанні, лише активному гравцю і лише у фазі PLACE_TILE.
export function highlightedCells(game: GameView, youPlayerId: string, rotation: Rotation): Cell[] {
  if (game.currentPlayerId !== youPlayerId || game.phase !== 'PLACE_TILE') return []
  return game.legalPlacements.filter((p) => p.rotations.includes(rotation)).map(({ x, y }) => ({ x, y }))
}
