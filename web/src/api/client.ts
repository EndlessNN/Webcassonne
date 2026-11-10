import type {
  ApiError,
  CreateRoomResponse,
  HealthResponse,
  JoinRoomResponse,
  RoomPreview,
  ServerMessage,
} from './types'

// Помилка API у єдиному форматі docs/api.md §1.
export class RequestError extends Error {
  code: string
  constructor(code: string, message: string) {
    super(message)
    this.code = code
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, init)
  const body: unknown = await res.json()
  if (!res.ok) {
    const { error } = body as ApiError
    throw new RequestError(error.code, error.message)
  }
  return body as T
}

const postJSON = (body: unknown): RequestInit => ({
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body),
})

export const getHealth = () => request<HealthResponse>('/api/health')

export const createRoom = (nickname: string) =>
  request<CreateRoomResponse>('/api/rooms', postJSON({ nickname }))

export const getRoomPreview = (code: string) =>
  request<RoomPreview>(`/api/rooms/${encodeURIComponent(code)}`)

export const joinRoom = (code: string, nickname: string) =>
  request<JoinRoomResponse>(`/api/rooms/${encodeURIComponent(code)}/players`, postJSON({ nickname }))

export function openRoomSocket(code: string, token: string, onMessage: (m: ServerMessage) => void): WebSocket {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const ws = new WebSocket(
    `${proto}//${location.host}/api/rooms/${encodeURIComponent(code)}/ws?token=${encodeURIComponent(token)}`,
  )
  ws.onmessage = (e: MessageEvent<string>) => onMessage(JSON.parse(e.data) as ServerMessage)
  return ws
}

export const inviteLink = (code: string) => `${location.origin}/r/${code}`
