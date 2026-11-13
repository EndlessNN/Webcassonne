import { useEffect, useState } from 'react'
import { getHealth } from './api/client'
import Home from './pages/Home'
import Invite from './pages/Invite'
import Lobby from './pages/Lobby'

export interface Session {
  roomCode: string
  token: string // лише в пам'яті вкладки (ADR-0005)
}

const inviteCode = /^\/r\/([A-Za-z0-9]+)$/.exec(location.pathname)?.[1]

export default function App() {
  const [session, setSession] = useState<Session | null>(null)
  const [serverDown, setServerDown] = useState(false)

  useEffect(() => {
    getHealth().catch(() => setServerDown(true))
  }, [])

  function enter(s: Session) {
    history.replaceState(null, '', `/r/${s.roomCode}`)
    setSession(s)
  }

  return (
    <main>
      <h1>
        <a href="/">Webcassonne</a>
      </h1>
      {serverDown && <p className="fail">Сервер недоступний. Запустіть go -C server run ./cmd/server</p>}
      {session ? (
        <Lobby session={session} />
      ) : inviteCode ? (
        <Invite code={inviteCode.toUpperCase()} onEnter={enter} />
      ) : (
        <Home onEnter={enter} />
      )}
    </main>
  )
}
