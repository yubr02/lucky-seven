import { useEffect, useRef, useState } from 'react'

export function useGameEffects() {
  const [sound, setSound] = useState(false)
  const [revealed, setRevealed] = useState(3)
  const audio = useRef<AudioContext | null>(null)
  const enabled = useRef(false)
  const alive = useRef(true)
  const waits = useRef(new Map<number, () => void>())

  useEffect(() => {
    alive.current = true
    return () => {
      alive.current = false
      for (const [timer, resolve] of waits.current) { window.clearTimeout(timer); resolve() }
      waits.current.clear()
      void audio.current?.close().catch(() => {})
      audio.current = null
    }
  }, [])

  async function toggleSound() {
    if (enabled.current) { enabled.current = false; setSound(false); return }
    try {
      audio.current ??= new AudioContext()
      await audio.current.resume()
      if (alive.current) { enabled.current = true; setSound(true) }
    } catch { enabled.current = false; setSound(false) }
  }

  function tone(frequency: number, duration = .1) {
    const ctx = audio.current
    if (!enabled.current || !ctx || ctx.state !== 'running') return
    try {
      const oscillator = ctx.createOscillator()
      const gain = ctx.createGain()
      oscillator.type = 'sine'
      oscillator.frequency.value = frequency
      gain.gain.setValueAtTime(.045, ctx.currentTime)
      gain.gain.exponentialRampToValueAtTime(.001, ctx.currentTime + duration)
      oscillator.connect(gain); gain.connect(ctx.destination)
      oscillator.start(); oscillator.stop(ctx.currentTime + duration)
      oscillator.onended = () => { oscillator.disconnect(); gain.disconnect() }
    } catch { /* Audio must never interrupt a confirmed play. */ }
  }

  function wait(ms: number) {
    return new Promise<void>(resolve => {
      const timer = window.setTimeout(() => { waits.current.delete(timer); resolve() }, ms)
      waits.current.set(timer, resolve)
    })
  }

  async function reveal(multiplier: number) {
    if (!window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      for (let index = 1; index <= 3; index++) {
        await wait(index === 1 ? 280 : 240)
        if (!alive.current) return false
        setRevealed(index)
        tone(300 + index * 140)
      }
    } else setRevealed(3)
    if (multiplier > 0) tone(multiplier === 100 ? 1046 : multiplier === 20 ? 880 : 660, .35)
    return alive.current
  }

  return { sound, toggleSound, revealed, startReveal: () => setRevealed(0), reveal }
}
