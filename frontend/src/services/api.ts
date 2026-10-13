export type Play = {
  id: number
  numbers: [number, number, number]
  bet: number
  payout: number
  balance: number
  created_at: string
}

export class ApiError extends Error {
  constructor(public status: number, message: string) { super(message) }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const controller = new AbortController()
  const timeout = window.setTimeout(() => controller.abort(), 12000)
  try {
    const response = await fetch(path, { ...options, signal: controller.signal })
    if (!response.ok) {
      const body = await response.json().catch(() => ({}))
      const messages: Record<string, string> = {
        insufficient_balance: 'Saldo insuficiente para esta aposta.',
        invalid_request: 'Use uma aposta inteira entre 1 e 100 créditos.',
        idempotency_conflict: 'Esta tentativa já foi usada com outro valor. Atualize o histórico antes de jogar novamente.',
      }
      throw new ApiError(response.status, messages[body.error] || `Não foi possível concluir a solicitação (HTTP ${response.status}).`)
    }
    return await response.json() as T
  } catch (error) {
    if (error instanceof ApiError) throw error
    throw new Error('Não foi possível obter a resposta do servidor. Verifique sua conexão e tente novamente.')
  } finally { window.clearTimeout(timeout) }
}

export const getBalance = () => request<{ balance: number }>('/api/balance')
export const getHistory = (before?: number) => request<Play[]>(`/api/history${before ? `?before=${before}` : ''}`)
export const play = (bet: number, key: string) => request<Play>('/api/play', {
  method: 'POST', headers: { 'Content-Type': 'application/json', 'Idempotency-Key': key }, body: JSON.stringify({ bet }),
})
export async function checkApiHealth(): Promise<void> {
  const result = await request<{ status: string }>('/health')
  if (result.status !== 'ok') throw new Error('Servidor indisponível.')
}
