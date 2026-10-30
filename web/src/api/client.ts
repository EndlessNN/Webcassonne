import type { ApiError, HealthResponse } from './types'

export async function getHealth(): Promise<HealthResponse> {
  const res = await fetch('/api/health')
  const body: unknown = await res.json()
  if (!res.ok) throw new Error((body as ApiError).error.message)
  return body as HealthResponse
}
