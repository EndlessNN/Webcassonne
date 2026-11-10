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

export interface RoomView {
  code: string
  status: RoomStatus
  hostPlayerId: string
  youPlayerId: string
  players: PlayerView[]
  game: null // shortcut: GameView з docs/api.md додається разом з US-05
}

export type ServerMessage =
  | { type: 'room_state'; room: RoomView }
  | ({ type: 'error' } & ApiError)
