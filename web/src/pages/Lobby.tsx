import { useEffect, useRef, useState } from 'react'
import { inviteLink, openRoomSocket } from '../api/client'
import type { RoomView } from '../api/types'
import type { Session } from '../App'

// US-02 (посилання) і US-04 (живий список гравців).
export default function Lobby({ session }: { session: Session }) {
  const [room, setRoom] = useState<RoomView | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [closed, setClosed] = useState(false)

  useEffect(() => {
    const ws = openRoomSocket(session.roomCode, session.token, (m) => {
      if (m.type === 'room_state') setRoom(m.room)
      else setError(m.error.message)
    })
    ws.onclose = () => setClosed(true)
    return () => {
      ws.onclose = null
      ws.close()
    }
  }, [session])

  if (closed) return <p className="fail">З’єднання з кімнатою втрачено. Створіть нову кімнату.</p>
  if (!room) return <p>Підключаємося до кімнати…</p>

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
