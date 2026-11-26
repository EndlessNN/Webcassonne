// Типи з docs/api.md. Змінюються лише разом з документом.

export interface ApiError {
  error: {
    code: string
    message: string
    details?: Record<string, unknown>
  }
}

export interface HealthResponse {
  status: 'ok'
  storage: { type: 'memory'; rooms: number }
}

export type RoomStatus = 'LOBBY' | 'PLAYING' | 'FINISHED' | 'ABANDONED'
export type Color = 'RED' | 'BLUE' | 'GREEN' | 'YELLOW' | 'BLACK'

export interface CreateRoomResponse {
  roomCode: string
  playerId: string
  token: string
}

export interface JoinRoomResponse {
  playerId: string
  token: string
}

export interface RoomPreview {
  code: string
  status: RoomStatus
  playerCount: number
  maxPlayers: number
}

export interface PlayerView {
  id: string
  nickname: string
  color: Color | null
  seat: number | null
  meeplesInSupply: number
  score: number
}

export type TurnPhase = 'PLACE_TILE' | 'PLACE_MEEPLE'
export type Rotation = 0 | 90 | 180 | 270
export type FeatureType = 'ROAD' | 'CITY' | 'FIELD' | 'MONASTERY'

export interface TileRef {
  tileTypeId: string
}

export interface LegalPlacement {
  x: number
  y: number
  rotations: Rotation[]
}

export interface PlacedTileView {
  x: number
  y: number
  tileTypeId: string
  rotation: Rotation
}

export interface MeepleView {
  playerId: string
  x: number
  y: number
  segment: number
}

export interface FinalScore {
  playerId: string
  road: number
  city: number
  monastery: number
  field: number
  total: number
  winner: boolean
}

export interface GameView {
  phase: TurnPhase
  turnNumber: number
  currentPlayerId: string
  deckCount: number
  currentTile: TileRef | null
  discardedTiles: TileRef[]
  legalPlacements: LegalPlacement[]
  legalMeepleSegments: number[]
  tiles: PlacedTileView[]
  meeples: MeepleView[]
  finalScores: FinalScore[] | null
}

export interface RoomView {
  code: string
  status: RoomStatus
  hostPlayerId: string
  youPlayerId: string
  players: PlayerView[]
  game: GameView | null // null у LOBBY
}

// GET /api/tiles — каталог типів тайлів (TileType з docs/data-model.md).
export interface Segment {
  type: FeatureType
  points: string[] // точки країв, напр. "N0"; порожній для монастиря
  pennant: boolean
  adjacentCities: number[]
}

export interface TileType {
  id: string
  count: number
  isStart: boolean
  segments: Segment[]
}

export interface TilesResponse {
  tiles: TileType[]
}

export type ClientMessage =
  | { type: 'start_game' }
  | { type: 'place_tile'; x: number; y: number; rotation: Rotation }
  | { type: 'place_meeple'; segment: number }
  | { type: 'skip_meeple' }

export type ServerMessage =
  | { type: 'room_state'; room: RoomView }
  | ({ type: 'error' } & ApiError)
