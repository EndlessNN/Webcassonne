import { useCallback, useEffect, useRef, useState } from 'react'
import { inviteLink, openRoomSocket, sendMessage } from '../api/client'
import type { ClientMessage, RoomView } from '../api/types'
import type { Session } from '../App'
import Game from './Game'

// US-02 (посилання), US-04 (живий список гравців), US-05 (старт партії). Після старту той самий сокет обслуговує екран гри.
export default function Lobby({ session }: { session: Session }) {
  const [room, setRoom] = useState<RoomView | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [closed, setClosed] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    const ws = openRoomSocket(session.roomCode, session.token, (m) => {
      if (m.type === 'room_state') {
        setRoom(m.room)
        setError(null) // стан змінився — попередня помилка дії вже неактуальна
      } else {
        setError(m.error.message)
      }
    })
    ws.onclose = () => setClosed(true)
    wsRef.current = ws
    return () => {
      ws.onclose = null
      ws.close()
      wsRef.current = null
    }
  }, [session])

  const send = useCallback((m: ClientMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) sendMessage(wsRef.current, m)
  }, [])

  if (closed) return <p className="fail">З’єднання з кімнатою втрачено. Створіть нову кімнату.</p>
  if (!room) return <p>Підключаємося до кімнати…</p>
  if (room.status !== 'LOBBY' && room.game) return <Game room={room} game={room.game} send={send} error={error} />

  const isHost = room.youPlayerId === room.hostPlayerId
  const enough = room.players.length >= 2

  return (
    <>
      <h2>Лобі кімнати {room.code}</h2>
      <InviteLink code={room.code} />
      <h3>Гравці ({room.players.length} з 5)</h3>
      <ul>
        {room.players.map((p) => (
          <li key={p.id}>
            {p.nickname}
            {p.id === room.hostPlayerId && ' — організатор'}
            {p.id === room.youPlayerId && ' (ви)'}
          </li>
        ))}
      </ul>
      {isHost ? (
        <div>
          <button disabled={!enough} onClick={() => send({ type: 'start_game' })}>
            Почати гру
          </button>
          {!enough && <span className="hint"> Потрібно щонайменше 2 гравці</span>}
        </div>
      ) : (
        <p>Очікуємо, поки організатор почне гру</p>
      )}
      {error && <p className="fail">{error}</p>}
    </>
  )
}

function InviteLink({ code }: { code: string }) {
  const link = inviteLink(code)
  const [copied, setCopied] = useState(false)
  const [manual, setManual] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!copied) return
    const t = setTimeout(() => setCopied(false), 2000)
    return () => clearTimeout(t)
  }, [copied])

  useEffect(() => {
    if (manual) inputRef.current?.select()
  }, [manual])

  async function copy() {
    try {
      await navigator.clipboard.writeText(link)
      setCopied(true)
    } catch {
      setManual(true) // буфер обміну недоступний: показуємо поле для ручного копіювання
    }
  }

  return (
    <div>
      <button onClick={copy}>Скопіювати посилання</button>
      {copied && <span className="ok"> Скопійовано</span>}
      {manual && (
        <div>
          <label htmlFor="invite-link">Скопіюйте посилання вручну: </label>
          <input id="invite-link" ref={inputRef} value={link} readOnly size={40} />
        </div>
      )}
    </div>
  )
}
