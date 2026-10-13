import { useEffect, useRef, useState } from 'react'
import { ApiError, getBalance, getHistory, play, type Play } from '../../services/api'
import { useGameEffects } from './useGameEffects'
import History from './History'
import Demonstration from './Demonstration'
import { useLiveUpdates } from './useLiveUpdates'

type Pending = { bet: number; key: string }
const pendingKey = 'lucky-seven.pending.v1'
const credits = (value: number) => value.toLocaleString('pt-BR')
const message = (error: unknown) => error instanceof Error ? error.message : 'Não foi possível concluir a operação.'
function readPending(): Pending | null {
  try {
    const value = JSON.parse(localStorage.getItem(pendingKey) || 'null') as Pending | null
    return value && Number.isInteger(value.bet) && value.bet >= 1 && value.bet <= 100 && typeof value.key === 'string' && /^[a-zA-Z0-9_-]{1,128}$/.test(value.key) ? value : null
  } catch { return null }
}

export default function StartScreen() {
  const [balance, setBalance] = useState<number | null>(null)
  const [history, setHistory] = useState<Play[]>([])
  const [bet, setBet] = useState('10')
  const [result, setResult] = useState<Play | null>(null)
  const [pending, setPending] = useState<Pending | null>(readPending)
  const [busy, setBusy] = useState(false)
  const [loading, setLoading] = useState(true)
  const [more, setMore] = useState(false)
  const [error, setError] = useState('')
  const [syncError, setSyncError] = useState('')
  const lock = useRef(false)
  const effects = useGameEffects()
  const live = useLiveUpdates()

  useEffect(() => {
    // Defer remote refreshes during a local play/animation or history request.
    if (!live.revision || busy || loading) return
    let active = true
    Promise.all([getBalance(), getHistory()]).then(([wallet, rows]) => {
      if (!active) return
      setBalance(wallet.balance)
      setHistory(previous => [...rows, ...previous.filter(row => rows.length > 0 && row.id < rows[rows.length - 1].id)])
      if (history.length < 50) setMore(rows.length === 50)
      setSyncError('')
    }).catch((err: unknown) => { if (active) setSyncError(message(err)) })
    return () => { active = false }
  }, [live.revision, busy, loading])

  async function refresh() {
    const [wallet, rows] = await Promise.all([getBalance(), getHistory()])
    setBalance(wallet.balance)
    setHistory(rows)
    setMore(rows.length === 50)
    setSyncError('')
  }
  useEffect(() => {
    let active = true
    Promise.all([getBalance(), getHistory()]).then(([wallet, rows]) => {
      if (!active) return
      setBalance(wallet.balance); setHistory(rows); setMore(rows.length === 50)
    }).catch((err: unknown) => { if (active) setSyncError(message(err)) })
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [])

  const amount = Number(bet)
  const valid = bet.trim() !== '' && Number.isInteger(amount) && amount >= 1 && amount <= 100
  const insufficient = balance !== null && valid && amount > balance
  const totalBet = history.reduce((sum, row) => sum + row.bet, 0)
  const totalPayout = history.reduce((sum, row) => sum + row.payout, 0)
  const winningPlays = history.filter(row => row.payout > 0).length
  const multiplier = result ? result.payout / result.bet : 0
  const resultKind = multiplier === 100 ? 'jackpot' : multiplier === 20 ? 'triple' : multiplier === 2 ? 'pair' : 'loss'
  const resultTitle = { jackpot: '777! O encontro dos setes.', triple: 'Três iguais. Bela combinação!', pair: 'Um par. Você ganhou!', loss: 'Desta vez, sem prêmio.' }[resultKind]

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    if (lock.current || loading || (!pending && (!valid || insufficient || balance === null))) return
    lock.current = true; setBusy(true); setError(''); setResult(null); effects.startReveal()
    let attempt = pending
    try {
      if (!attempt) {
        attempt = { bet: amount, key: crypto.randomUUID() }
        // Persist before sending so a reload can recover an uncertain response.
        localStorage.setItem(pendingKey, JSON.stringify(attempt))
        setPending(attempt)
      }
      const outcome = await play(attempt.bet, attempt.key)
      setResult(outcome)
      if (!await effects.reveal(outcome.payout / outcome.bet)) return
      setBalance(outcome.balance)
      setHistory(rows => [outcome, ...rows.filter(row => row.id !== outcome.id)].sort((a, b) => b.id - a.id))
      localStorage.removeItem(pendingKey)
      setPending(null)
      await refresh().catch((err: unknown) => setSyncError(`Jogada confirmada. Falha ao atualizar saldo e histórico: ${message(err)}`))
    } catch (err) {
      if (err instanceof ApiError && [400, 409, 422].includes(err.status)) {
        localStorage.removeItem(pendingKey); setPending(null)
        await refresh().catch((sync: unknown) => setSyncError(message(sync)))
      }
      setError(message(err))
    } finally { lock.current = false; setBusy(false) }
  }

  async function updateHistory(older: boolean) {
    if (lock.current) return
    lock.current = true; setLoading(true)
    try {
      if (older && history.length) {
        const rows = await getHistory(history[history.length - 1].id)
        setHistory(current => [...current, ...rows.filter(row => !current.some(item => item.id === row.id))])
        setMore(rows.length === 50)
        setSyncError('')
      } else await refresh()
    } catch (err) { setSyncError(message(err)) }
    finally { lock.current = false; setLoading(false) }
  }

  return (
    <main className="game-page" id="inicio">
      <a className="skip-link" href="#jogar">Pular para o jogo</a>
      <header className="topbar">
        <a href="#inicio" className="brand" aria-label="Lucky Seven, início"><span className="brand-mark">7</span><span>LUCKY<br />SEVEN<span className="brand-caption">PLAYGROUND</span></span></a>
        <nav aria-label="Navegação principal"><a href="#jogar">O jogo</a><a href="#carteira">Carteira</a><a href="#historico">Histórico</a></nav>
        <a className="header-cta" href="#jogar">Jogar demo <span aria-hidden="true">↗</span></a>
      </header>
      <section className="play-heading" id="jogar"><div><p className="eyebrow">TRÊS NÚMEROS. SUA PRÓXIMA COMBINAÇÃO.</p><h1>Lucky <em>Seven.</em><span className="live-badge">DEMO</span></h1><p>Escolha sua aposta e jogue. Apenas créditos fictícios.</p></div><div className="wallet" id="carteira"><span>SUA CARTEIRA</span><strong>{balance === null ? '—' : credits(balance)} <small>CR</small></strong><span>créditos fictícios disponíveis</span></div></section>
      <div className="game-layout">
        <section className={`machine ${result && !busy ? `machine--${resultKind}` : ''}`} aria-labelledby="machine-title">
          <div className="live-status" role="status">{live.connected ? '● Atualizações em tempo real conectadas' : '○ Reconectando atualizações · você pode continuar jogando'}</div>
          <div className="machine-heading"><h2 id="machine-title">Sua próxima jogada</h2><button className="sound-toggle" type="button" aria-pressed={effects.sound} onClick={() => void effects.toggleSound()}>{effects.sound ? '♪ Som ligado' : '♪ Som desligado'}</button></div>
          <div className="reels" aria-label={busy ? 'Revelando os números da jogada' : result ? `Números sorteados: ${result.numbers.join(', ')}` : 'Aguardando a primeira jogada'} aria-busy={busy}>
            {(result?.numbers ?? ['?', '?', '?']).map((number, index) => {
              const hidden = busy && index >= effects.revealed
              const matched = result && result.payout > 0 && result.numbers.filter(n => n === number).length > 1
              return <div className={`reel ${hidden ? 'reel--rolling' : ''} ${!hidden && number === 7 ? 'reel--seven' : ''} ${!busy && matched ? 'reel--matched' : ''}`} key={index} aria-hidden="true"><span>{hidden ? '✦' : number}</span><i>✦</i></div>
            })}
          </div>
          <div className={`result ${result && !busy ? `result--${resultKind}` : ''}`} role="status" aria-live="polite" aria-atomic="true">
            {busy ? <><strong>{result ? 'Revelando sua combinação…' : 'Confirmando sua jogada…'}</strong><span>Aguarde o resultado.</span></> : result ? <><div className="result-heading"><strong>{resultTitle}</strong>{multiplier > 0 && <b className="result-multiplier">{multiplier}×</b>}</div><div className="result-breakdown"><div><span>Aposta</span><strong>{credits(result.bet)} <small>CR</small></strong></div><div><span>Retorno</span><strong>{credits(result.payout)} <small>CR</small></strong></div><div><span>Variação do saldo</span><strong>{result.payout - result.bet > 0 ? '+' : ''}{credits(result.payout - result.bet)} <small>CR</small></strong></div></div></> : <><strong>{pending ? 'Sua jogada pode ser recuperada.' : 'Tudo pronto para jogar.'}</strong><span>{pending ? 'Use o botão abaixo para consultar o resultado.' : 'Aposte de 1 a 100 créditos.'}</span></>}
          </div>
          <form onSubmit={submit} className="bet-form">
            <label htmlFor="bet">Sua aposta <span>em créditos</span></label>
            <div className="bet-controls"><input id="bet" type="number" inputMode="numeric" min="1" max="100" step="1" value={pending ? pending.bet : bet} onChange={event => setBet(event.target.value)} disabled={busy || !!pending} aria-describedby="bet-help" required /><button className="play-button" disabled={busy || loading || (!pending && (!valid || insufficient || balance === null))}>{busy ? 'Aguarde…' : pending ? 'Recuperar jogada' : 'Jogar agora'} <span aria-hidden="true">↗</span></button></div>
            <div className="quick-bets" aria-label="Valores de aposta">{[1, 10, 25, 50, 100].map(value => <button type="button" key={value} aria-pressed={!pending && amount === value} onClick={() => setBet(String(value))} disabled={busy || !!pending}>{value}</button>)}</div>
            <p id="bet-help" className={insufficient && !pending ? 'field-help field-help--error' : 'field-help'}>{pending ? 'Há uma jogada aguardando confirmação. Recupere o resultado antes de fazer outra aposta.' : insufficient ? 'Saldo insuficiente. Escolha um valor menor.' : !valid ? 'Digite um número inteiro entre 1 e 100.' : 'O valor é debitado uma única vez por jogada.'}</p>
          </form>
          {error && <p className="error-message" role="alert">{error}</p>}
        </section>
        <aside className="rules" id="regras"><p className="eyebrow">CONHEÇA AS COMBINAÇÕES</p><h2>Os números mudam.<br />As regras, não.</h2><div className="prize prize--jackpot"><span className="combination">7 7 7</span><div><strong>100×</strong><span>a aposta</span></div></div><div className="prize"><div><strong>Três iguais</strong><span>Exceto 777</span></div><b>20×</b></div><div className="prize"><div><strong>Dois iguais</strong><span>Em qualquer posição</span></div><b>2×</b></div><div className="prize"><div><strong>Todos diferentes</strong><span>Sem retorno nesta jogada</span></div><b>0×</b></div><p className="rules-note">Números de 0 a 9, com repetição. O retorno inclui o valor apostado. Os prêmios não se acumulam.</p><div className="demo-note"><span>✳</span><p>Só pela diversão.<br /><strong>Sem dinheiro real.</strong></p></div></aside>
      </div>
      <section className="activity" aria-labelledby="activity-title"><div className="activity-heading"><p className="eyebrow">02 / TUDO CONECTADO</p><h2 id="activity-title">Seu jogo, em perspectiva.</h2><p>Resumo das {history.length} jogadas carregadas no histórico.</p></div><div className="activity-grid"><article><span>JOGADAS CARREGADAS</span><strong>{history.length}</strong><p>Cada resultado registrado.</p></article><article><span>CRÉDITOS APOSTADOS</span><strong>{credits(totalBet)}<small> CR</small></strong><p>Movimentação do seu saldo fictício.</p></article><article><span>RETORNOS RECEBIDOS</span><strong>{credits(totalPayout)}<small> CR</small></strong><p>{winningPlays} {winningPlays === 1 ? 'jogada com retorno' : 'jogadas com retorno'}.</p></article></div></section>
      <History rows={history} loading={loading} busy={busy} more={more} error={syncError} refresh={() => void updateHistory(false)} loadMore={() => void updateHistory(true)} />
      <Demonstration />
      <footer><a href="#inicio" className="footer-brand">LUCKY SEVEN<span>PLAYGROUND</span></a><p>Projeto de portfólio inspirado na Jungle Gaming.<br />Demonstração independente com créditos fictícios.</p><a href="https://junglegaming.io" target="_blank" rel="noreferrer">Conheça a inspiração ↗</a></footer>
    </main>
  )
}
