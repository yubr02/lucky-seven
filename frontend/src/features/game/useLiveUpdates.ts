import { useEffect, useState } from 'react'

export function useLiveUpdates() {
  const [revision, setRevision] = useState(0)
  const [connected, setConnected] = useState(false)
  useEffect(() => {
    let stopped = false
    let socket: WebSocket | null = null
    let timer: number | undefined
    let delay = 1000
    function connect() {
      if (stopped) return
      const url = new URL('/api/events', window.location.href)
      url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
      const current = new WebSocket(url)
      socket = current
      current.onmessage = event => {
        if (stopped || socket !== current) return
        try {
          const message = JSON.parse(event.data) as { type?: string }
          if (message.type === 'ready' || message.type === 'wallet.changed') {
            delay = 1000
            setConnected(true)
            setRevision(value => value + 1)
          }
        } catch { /* Ignore messages outside the notification contract. */ }
      }
      current.onerror = () => current.close()
      current.onclose = () => {
        if (stopped || socket !== current) return
        setConnected(false)
        timer = window.setTimeout(connect, delay)
        delay = Math.min(delay * 2, 15000)
      }
    }
    const onVisible = () => {
      if (document.visibilityState === 'visible') setRevision(value => value + 1)
    }
    connect()
    document.addEventListener('visibilitychange', onVisible)
    return () => {
      stopped = true
      window.clearTimeout(timer)
      socket?.close()
      document.removeEventListener('visibilitychange', onVisible)
    }
  }, [])
  return { revision, connected }
}
