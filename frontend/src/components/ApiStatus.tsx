type ApiStatusProps = {
  state: 'loading' | 'success' | 'error'
  message?: string
}

const statusLabels = {
  loading: 'Consultando a API...',
  success: 'API ativa',
  error: 'API indisponível',
}

export default function ApiStatus({ state, message }: ApiStatusProps) {
  return (
    <div className={`api-status api-status--${state}`} role="status" aria-live="polite">
      <span className="api-status__indicator" aria-hidden="true" />
      <span>{message ?? statusLabels[state]}</span>
    </div>
  )
}