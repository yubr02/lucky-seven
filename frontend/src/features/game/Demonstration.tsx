import { useRef, useState } from 'react'
import { useGameEffects } from './useGameEffects'

const scenarios = [
  { id: 'loss', label: 'Sem prêmio', numbers: [1, 2, 3], payout: 0 },
  { id: 'pair', label: 'Dois iguais', numbers: [2, 4, 2], payout: 20 },
  { id: 'triple', label: 'Três iguais', numbers: [4, 4, 4], payout: 200 },
  { id: 'jackpot', label: '777', numbers: [7, 7, 7], payout: 1000 },
]
type Example = typeof scenarios[number]

export default function Demonstration() {
  const [scenario, setScenario] = useState('jackpot')
  const [result, setResult] = useState<Example | null>(null)
  const [state, setState] = useState<'idle' | 'busy' | 'done' | 'insufficient' | 'network'>('idle')
  const effects = useGameEffects()
  const lock = useRef(false)

  async function run(recover = false) {
    if (lock.current) return
    lock.current = true
    setResult(null)
    if (scenario === 'insufficient') { setState('insufficient'); lock.current = false; return }
    if (scenario === 'network' && !recover) { setState('network'); lock.current = false; return }
    setState('busy'); effects.startReveal()
    const example = scenarios.find(item => item.id === scenario) ?? scenarios[1]
    setResult(example)
    try { if (await effects.reveal(example.payout / 10)) setState('done') }
    finally { lock.current = false }
  }

  return <details className="demonstration" id="demonstracao">
    <summary><span>Laboratório de demonstração</span><small>Exemplos sem alterar sua carteira</small></summary>
    <div className="demo-content"><p className="simulation-label">SIMULAÇÃO LOCAL · SEM APOSTA OU REGISTRO NO HISTÓRICO</p><p>Explore os resultados e situações de erro. Todos os valores abaixo são exemplos com aposta de 10 créditos e saldo inicial de 1.000, exceto no cenário de saldo insuficiente.</p>
      <div className="demo-controls"><div className="demo-field"><label htmlFor="scenario">Cenário</label><select id="scenario" value={scenario} disabled={state === 'busy'} onChange={event => { setScenario(event.target.value); setResult(null); setState('idle') }}>{scenarios.map(item => <option key={item.id} value={item.id}>{item.label}</option>)}<option value="insufficient">Saldo insuficiente</option><option value="network">Resposta perdida e recuperação</option></select></div><button className="text-button" disabled={state === 'busy'} onClick={() => void run()}>{state === 'busy' ? 'Demonstrando…' : 'Visualizar exemplo'}</button></div>
      {result && <div className={`demo-preview machine--${state === 'done' ? result.id : 'pending'}`}><div className="detail-numbers" aria-hidden="true">{result.numbers.map((n, i) => <b className={i >= effects.revealed && state === 'busy' ? 'reel--rolling' : ''} key={i}><span>{i < effects.revealed ? n : '✦'}</span></b>)}</div><div role="status" aria-live="polite">{state === 'done' ? <><strong>{result.label} · {result.numbers.join(' / ')} · {result.payout / 10}×</strong><p>Aposta: 10 CR · Retorno: {result.payout} CR · Saldo simulado: {1000 - 10 + result.payout} CR</p>{scenario === 'network' && <p>Exemplo recuperado: a mesma jogada, com um único débito de 10 CR. Nenhuma chamada foi enviada ao servidor.</p>}</> : 'Revelando o exemplo…'}</div></div>}
      {state === 'insufficient' && <div className="error-message" role="status"><strong>Exemplo: saldo insuficiente.</strong><p>Saldo disponível: 5 CR. Aposta solicitada: 10 CR. A jogada seria recusada antes do sorteio, sem débito. Saldo final do exemplo: 5 CR.</p></div>}
      {state === 'network' && <div className="error-message" role="status"><strong>Exemplo: a resposta não chegou.</strong><p>O servidor pode ter concluído a jogada. A recuperação usa a mesma chave para consultar o resultado sem cobrar novamente.</p><button className="text-button" onClick={() => void run(true)}>Simular recuperação</button></div>}
    </div>
  </details>
}
