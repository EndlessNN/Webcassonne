import { useEffect, useState } from 'react'
import { getRoomPreview, joinRoom } from '../api/client'
import type { RoomPreview } from '../api/types'
import type { Session } from '../App'
import NicknameForm from './NicknameForm'

// US-03: вхід у кімнату за посиланням /r/{code}.
export default function Invite({ code, onEnter }: { code: string; onEnter: (s: Session) => void }) {
  const [preview, setPreview] = useState<RoomPreview | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getRoomPreview(code).then(setPreview, (e: Error) => setError(e.message))
  }, [code])

  if (error) {
    return (
      <>
        <p className="fail">{error}</p>
        <a href="/">На головну</a>
      </>
    )
  }
  if (!preview) return <p>Завантажуємо кімнату…</p>
  if (preview.status !== 'LOBBY') return <p className="fail">Партія вже почалася</p>
  if (preview.playerCount >= preview.maxPlayers) {
    return <p className="fail">Кімната заповнена (максимум {preview.maxPlayers} гравців)</p>
  }

  return (
    <>
      <p>
        Вас запросили в кімнату <b>{code}</b> ({preview.playerCount} з {preview.maxPlayers} гравців).
      </p>
      <NicknameForm
        submitLabel="Приєднатися"
        onSubmit={async (nickname) => {
          const res = await joinRoom(code, nickname)
          onEnter({ roomCode: code, token: res.token })
        }}
      />
    </>
  )
}
