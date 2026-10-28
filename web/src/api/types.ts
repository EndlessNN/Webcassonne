// Типи з docs/api.md. Змінюються лише разом з документом.

export interface ApiError {
  error: {
    code: string
    message: string
    details?: Record<string, unknown>
  }
}

export interface HealthResponse {
  status: 'ok'
  storage: { type: 'memory'; rooms: number }
}
