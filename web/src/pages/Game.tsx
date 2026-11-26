import { useEffect, useState } from 'react'
import { getTiles } from '../api/client'
import type { ClientMessage, GameView, PlayerView, Rotation, RoomView, TileType } from '../api/types'
import { PLAYER_COLORS } from '../game/colors'
import { highlightedCells } from '../game/highlight'
import { nextRotation } from '../game/rotation'
import Board from './Board'
import TileShape from './TileShape'

interface Props {
  room: RoomView
  game: GameView
  send: (m: ClientMessage) => void
  error: string | null
}

// Екран партії: US-06…US-16. Усе, що вирішує правила, приходить із сервера (legalPlacements, deckCount, score…).
export default function Game({ room, game, send, error }: Props) {
  const [catalog, setCatalog] = useState<Map<string, TileType> | null>(null)
  const [catalogError, setCatalogError] = useState<string | null>(null)
  // Обертання — локальна дія (US-10); для нового ходу тайл знову в початковій орієнтації.
  const [turnRotation, setTurnRotation] = useState<{ turn: number; rotation: Rotation }>({ turn: 0, rotation: 0 })
  const rotation: Rotation = turnRotation.turn === game.turnNumber ? turnRotation.rotation : 0

  useEffect(() => {
    getTiles()
      .then((r) => setCatalog(new Map(r.tiles.map((t) => [t.id, t]))))
      .catch((e: unknown) => setCatalogError(e instanceof Error ? e.message : 'Не вдалося завантажити тайли'))
  }, [])

  const playing = room.status === 'PLAYING'
  const myTurn = playing && game.currentPlayerId === room.youPlayerId
  const canRotate = myTurn && game.phase === 'PLACE_TILE'

  useEffect(() => {
    if (!canRotate) return
    const onKey = (e: KeyboardEvent) => {
      if ((e.key === 'r' || e.key === 'R' || e.key === 'к' || e.key === 'К') && !(e.target instanceof HTMLInputElement)) {
        setTurnRotation((prev) => ({
          turn: game.turnNumber,
          rotation: nextRotation(prev.turn === game.turnNumber ? prev.rotation : 0),
        }))
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [canRotate, game.turnNumber])

  if (catalogError) return <p className="fail">{catalogError}</p>
  if (!catalog) return <p>Завантаження тайлів…</p>

  const players = [...room.players].sort((a, b) => (a.seat ?? 0) - (b.seat ?? 0))
  const byId = new Map(room.players.map((p) => [p.id, p]))
  const current = byId.get(game.currentPlayerId)
  const currentType = game.currentTile ? catalog.get(game.currentTile.tileTypeId) : undefined
  const highlighted = playing ? highlightedCells(game, room.youPlayerId, rotation) : []
  const colorOf = (p: PlayerView | undefined) => (p?.color ? PLAYER_COLORS[p.color] : '#888')

  function rotate() {
    setTurnRotation({ turn: game.turnNumber, rotation: nextRotation(rotation) })
  }

  return (
    <div className="game">
      <div className="board-wrap">
        <Board
          tiles={game.tiles}
          meeples={game.meeples}
          catalog={catalog}
          meepleColor={(id) => colorOf(byId.get(id))}
          highlighted={highlighted}
          ghost={canRotate && currentType ? { type: currentType, rotation } : null}
          onCellClick={canRotate ? ({ x, y }) => send({ type: 'place_tile', x, y, rotation }) : null}
        />
        <p className="hint">Перетягуйте поле мишею, щоб побачити інші частини карти.</p>
      </div>

      <aside className="panel">
        {room.status === 'ABANDONED' && <p className="fail turn">Партію перервано</p>}
        {room.status === 'FINISHED' && <p className="turn">Партію завершено</p>}
        {playing && (
          <p className="turn">
            {myTurn ? (
              'Ваш хід'
            ) : (
              <>
                Хід: <span style={{ color: colorOf(current) }}>{current?.nickname}</span>
              </>
            )}
          </p>
        )}

        {playing && game.discardedTiles.length > 0 && <p className="note">Тайл неможливо покласти, береться новий</p>}

        {playing && currentType && game.phase === 'PLACE_TILE' && (
          <section>
            <h3>Поточний тайл</h3>
            <svg className="current-tile" viewBox="0 0 100 100">
              <g transform={`rotate(${myTurn ? rotation : 0} 50 50)`}>
                <TileShape type={currentType} />
              </g>
            </svg>
            {canRotate && (
              <div>
                <button onClick={rotate}>Повернути</button> <span className="hint">або клавіша R</span>
              </div>
            )}
          </section>
        )}

        {myTurn && game.phase === 'PLACE_MEEPLE' && (
          <section>
            <p>Тайл покладено.</p>
            <button onClick={() => send({ type: 'skip_meeple' })}>Без міпла</button>
          </section>
        )}

        <p>У стопці: {game.deckCount}</p>

        <table className="players">
          <thead>
            <tr>
              <th>#</th>
              <th>Гравець</th>
              <th>Міпли</th>
              <th>Очки</th>
            </tr>
          </thead>
          <tbody>
            {players.map((p) => (
              <tr key={p.id} className={playing && p.id === game.currentPlayerId ? 'current' : undefined}>
                <td>{(p.seat ?? 0) + 1}</td>
                <td>
                  <span className="swatch" style={{ background: colorOf(p) }} />
                  {p.nickname}
                  {p.id === room.youPlayerId && ' (ви)'}
                </td>
                <td>{p.meeplesInSupply}</td>
                <td>{p.score}</td>
              </tr>
            ))}
          </tbody>
        </table>

        {room.status === 'FINISHED' && game.finalScores && (
          <table className="players">
            <thead>
              <tr>
                <th>Гравець</th>
                <th>Дороги</th>
                <th>Міста</th>
                <th>Монастирі</th>
                <th>Поля</th>
                <th>Разом</th>
              </tr>
            </thead>
            <tbody>
              {game.finalScores.map((f) => (
                <tr key={f.playerId} className={f.winner ? 'current' : undefined}>
                  <td>
                    {byId.get(f.playerId)?.nickname}
                    {f.winner && ' 🏆'}
                  </td>
                  <td>{f.road}</td>
                  <td>{f.city}</td>
                  <td>{f.monastery}</td>
                  <td>{f.field}</td>
                  <td>{f.total}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        {error && <p className="fail">{error}</p>}
      </aside>
    </div>
  )
}
