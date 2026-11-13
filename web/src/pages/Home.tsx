import { createRoom } from '../api/client'
import type { Session } from '../App'
import NicknameForm from './NicknameForm'

// US-01: створення кімнати лише за нікнеймом.
export default function Home({ onEnter }: { onEnter: (s: Session) => void }) {
  return (
    <>
      <p>Створіть кімнату й надішліть друзям посилання. Реєстрація не потрібна.</p>
      <NicknameForm
        submitLabel="Створити кімнату"
        onSubmit={async (nickname) => {
          const res = await createRoom(nickname)
          onEnter({ roomCode: res.roomCode, token: res.token })
        }}
      />
    </>
  )
}
