import { useEffect, useState } from 'react'
import { getHealth } from './api/client'
import type { HealthResponse } from './api/types'

const screens = [
  ['Головна', 'створення кімнати за нікнеймом (US-01)'],
  ['Вхід за запрошенням', 'приєднання до кімнати (US-03)'],
  ['Лобі', 'список гравців, посилання, старт партії (US-02, US-04, US-05)'],
  ['Партія', 'поле, тайл, міпли, рахунок (US-06 – US-20)'],
  ['Результати', 'фінальна таблиця з переможцем (US-21 – US-23)'],
]

export default function App() {
  const [health, setHealth] = useState<HealthResponse | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getHealth().then(setHealth, (e: Error) => setError(e.message))
  }, [])

  return (
    <main>
      <h1>Webcassonne</h1>
      <p>Онлайн-версія базової Carcassonne для компанії друзів. Каркас без ігрової логіки.</p>

      <h2>Зв'язок із сервером</h2>
      {health && (
        <p className="ok">
          Сервер: {health.status}, сховище: {health.storage.type}, кімнат: {health.storage.rooms}
        </p>
      )}
      {error && <p className="fail">Сервер недоступний: {error}</p>}
      {!health && !error && <p>Перевіряємо /api/health…</p>}

      <h2>Майбутні екрани</h2>
      <ul>
        {screens.map(([name, desc]) => (
          <li key={name}>
            <b>{name}</b> — {desc}
          </li>
        ))}
      </ul>
    </main>
  )
}
