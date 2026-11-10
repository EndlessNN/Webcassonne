import { useState, type FormEvent } from 'react'

interface Props {
  submitLabel: string
  onSubmit: (nickname: string) => Promise<void>
}

// Форма нікнейму для US-01 і US-03. Валідацію робить сервер, форма показує його повідомлення.
export default function NicknameForm({ submitLabel, onSubmit }: Props) {
  const [nickname, setNickname] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    setError(null)
    try {
      await onSubmit(nickname)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Не вдалося зв’язатися з сервером')
      setBusy(false)
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <label htmlFor="nickname">Нікнейм</label>
      <div className="row">
        <input
          id="nickname"
          value={nickname}
          onChange={(e) => setNickname(e.target.value)}
          autoFocus
          aria-invalid={error !== null}
          aria-describedby={error ? 'nickname-error' : undefined}
        />
        <button type="submit" disabled={busy}>
          {submitLabel}
        </button>
      </div>
      {error && (
        <p id="nickname-error" className="fail" role="alert">
          {error}
        </p>
      )}
    </form>
  )
}
