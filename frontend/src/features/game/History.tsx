import { useEffect, useRef, useState } from 'react'
import type { Play } from '../../services/api'

const format = (n: number) => n.toLocaleString('pt-BR')
type Props = { rows: Play[]; loading: boolean; busy: boolean; more: boolean; error: string; refresh: () => void; loadMore: () => void }

function Details({ play, close }: { play: Play; close: () => void }) {
  const dialog = useRef<HTMLDialogElement>(null)
  useEffect(() => { dialog.current?.showModal() }, [])
  return <dialog ref={dialog} className="play-dialog" onClose={close} aria-labelledby="detail-title">
    <div className="section-heading"><h2 id="detail-title">Jogada #{play.id}</h2><button className="text-button" autoFocus onClick={() => dialog.current?.close()}>Fechar ×</button></div>
    <p>{new Date(play.created_at).toLocaleString('pt-BR')}</p>
    <div className="detail-numbers" aria-label={`Números: ${play.numbers.join(', ')}`}>{play.numbers.map((n, i) => <b key={i}>{n}</b>)}</div>
    <dl className="detail-list"><div><dt>Aposta</dt><dd>{format(play.bet)} CR</dd></div><div><dt>Retorno total</dt><dd>{format(play.payout)} CR</dd></div><div><dt>Multiplicador</dt><dd>{play.payout / play.bet}×</dd></div><div><dt>Saldo antes</dt><dd>{format(play.balance + play.bet - play.payout)} CR</dd></div><div><dt>Variação</dt><dd>{play.payout > play.bet ? '+' : ''}{format(play.payout - play.bet)} CR</dd></div><div><dt>Saldo após</dt><dd>{format(play.balance)} CR</dd></div></dl>
    <p className="field-help">O retorno inclui o valor apostado. Este é o saldo registrado naquela jogada.</p>
  </dialog>
}

export default function History({ rows, loading, busy, more, error, refresh, loadMore }: Props) {
  const [filter, setFilter] = useState('all')
  const [page, setPage] = useState(0)
  const [selected, setSelected] = useState<Play | null>(null)
  const filtered = rows.filter(row => filter === 'all' || (filter === 'wins' ? row.payout > 0 : row.payout === 0))
  const pages = Math.max(1, Math.ceil(filtered.length / 10))
  const current = Math.min(page, pages - 1)
  const visible = filtered.slice(current * 10, current * 10 + 10)
  // A new play or refresh returns the reader to the newest records.
  const newest = rows[0]?.id
  useEffect(() => { setPage(0) }, [newest])

  return <section className="history" id="historico" aria-labelledby="history-title">
    <div className="section-heading"><div><p className="eyebrow">03 / CADA JOGADA CONTA</p><h2 id="history-title">Seu histórico</h2></div><button className="text-button" disabled={loading || busy} onClick={() => { setPage(0); refresh() }}>{loading ? 'Atualizando…' : 'Atualizar ↻'}</button></div>
    <div className="history-filters" aria-label="Filtrar histórico">{[{ id: 'all', label: 'Todas' }, { id: 'wins', label: 'Premiadas' }, { id: 'losses', label: 'Sem prêmio' }].map(item => <button key={item.id} className="text-button" aria-pressed={filter === item.id} onClick={() => { setFilter(item.id); setPage(0) }}>{item.label}</button>)}</div>
    <p className="history-caption" role="status">{filtered.length} resultados entre {rows.length} jogadas carregadas.{more ? ' Há registros anteriores disponíveis.' : ''}</p>
    {error && <p className="error-message" role="alert">{error} Use “Atualizar” para tentar novamente.</p>}
    {loading && !rows.length ? <p className="empty-state">Carregando suas jogadas…</p> : !visible.length ? <p className="empty-state">{error ? 'Histórico indisponível no momento.' : rows.length ? 'Nenhuma jogada corresponde ao filtro nos registros carregados.' : 'Seu histórico começa na primeira jogada.'}</p> : <div className="table-scroll"><table><thead><tr><th>Jogada</th><th>Números</th><th>Aposta</th><th>Retorno</th><th>Saldo após</th><th>Detalhes</th></tr></thead><tbody>{visible.map(row => <tr key={row.id}><td><strong>#{row.id}</strong><time dateTime={row.created_at}>{new Date(row.created_at).toLocaleString('pt-BR')}</time></td><td><span className="mini-numbers">{row.numbers.map((n, i) => <span key={i}>{n}</span>)}</span></td><td>{format(row.bet)}</td><td className={row.payout > 0 ? 'winning' : ''}>{format(row.payout)}</td><td>{format(row.balance)}</td><td><button className="text-button" onClick={() => setSelected(row)} aria-label={`Ver detalhes da jogada ${row.id}`}>Ver ↗</button></td></tr>)}</tbody></table></div>}
    <div className="history-pagination"><button className="text-button" disabled={current === 0 || loading} onClick={() => setPage(current - 1)}>← Anterior</button><span>Página {current + 1} de {pages}</span><button className="text-button" disabled={current + 1 >= pages || loading} onClick={() => setPage(current + 1)}>Próxima →</button></div>
    {more && <button className="text-button load-more" disabled={busy || loading} onClick={loadMore}>{loading ? 'Carregando…' : 'Buscar mais registros anteriores ↓'}</button>}
    {selected && <Details play={selected} close={() => setSelected(null)} />}
  </section>
}
