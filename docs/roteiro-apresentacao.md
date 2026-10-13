# Roteiro de apresentação — Lucky Seven

Duração sugerida: 5 a 7 minutos. Demonstração nativa na máquina, sem hospedagem pública. Repositório: https://github.com/yubr02/lucky-seven.

## Preparação antes da entrevista

1. Siga [execução local](apresentacao-local.md) e abra o jogo em duas abas lado a lado.
2. Confira o saldo e faça uma aposta pequena. Use 1 crédito na apresentação.
3. Deixe abertos o README, `backend/internal/app/app.go`, `backend/internal/game/game.go`, `backend/internal/storage/postgres/store.go` e a página de Actions.
4. Tenha os testes já executados. Não dependa de instalar bibliotecas ou baixar imagens durante a chamada.
5. Não compartilhe o conteúdo de `.env`, tokens ou configurações da sua conta. Mostre o `.env.example` se precisar explicar configuração.

## 0:00–0:40 — Apresentação do projeto

> “Preparei o Lucky Seven como projeto de portfólio para praticar backend Go. É um jogo com créditos fictícios. Meu foco foi garantir consistência de saldo, evitar cobrança duplicada e organizar inicialização e encerramento dos recursos. A interface ajuda a demonstrar o fluxo.”

Não apresente o projeto como produto oficial da Jungle nem como sistema de produção. Explique o que você realmente entende; use estas falas como apoio, não como texto para decorar.

## 0:40–1:40 — Demonstração do jogo

- Mostre saldo, aposta e tabela de retornos.
- Faça uma aposta de 1 crédito. Mostre números, retorno e variação do saldo.
- Abra os detalhes dessa jogada no histórico. Explique que retorno inclui a aposta: saldo final = saldo anterior − aposta + retorno.
- Mostre que a segunda aba atualiza automaticamente.

> “O POST confirma a jogada no PostgreSQL antes de emitir o aviso WebSocket. A outra aba recebe uma notificação e consulta o estado atual pela API. O WebSocket não realiza apostas e não é a fonte de verdade do saldo.”

Se quiser mostrar 777, use o laboratório identificado como simulação. Não prometa obter um prêmio no sorteio real. O laboratório não chama a API nem altera o saldo.

## 1:40–3:00 — Percurso de uma requisição

Abra os arquivos nesta ordem:

1. `internal/http/router.go`: decodifica JSON, limita corpo, exige chave e traduz erros em status HTTP.
2. `internal/game/game.go`: valida aposta/chave e contém sorteio com `crypto/rand` e cálculo do prêmio.
3. `internal/storage/postgres/store.go`: executa a transação, bloqueia a carteira, consulta idempotência, verifica saldo, sorteia e persiste.

> “O bloqueio `SELECT ... FOR UPDATE` serializa jogadas da mesma carteira. A consulta da chave acontece antes da checagem de saldo. Se a jogada já existe, devolvo o resultado persistido. Para uma nova jogada, débito, prêmio e resultado são confirmados na mesma transação.”

Mostre `UNIQUE (player_id, idempotency_key)` na migração como proteção adicional. Explique que os locks são por carteira, não um bloqueio global de todos os jogadores.

### Exemplo opcional de idempotência

No PowerShell, com a API local em 8081:

```powershell
$key = [guid]::NewGuid().ToString()
$params = @{
    Uri = 'http://localhost:8081/api/play'
    Method = 'Post'
    ContentType = 'application/json'
    Headers = @{ 'Idempotency-Key' = $key }
    Body = '{"bet":1}'
}
Invoke-RestMethod @params
Invoke-RestMethod @params
```

As duas respostas têm o mesmo ID e resultado. A primeira chamada aposta 1 crédito; a segunda não cobra novamente. Uma aposta diferente com a mesma chave retorna 409. O saldo contido no replay é o da ocasião; `/api/balance` informa o saldo atual.

## 3:00–3:50 — Uber Fx

Abra `internal/app/app.go` e depois o `main.go`.

> “O Fx monta o grafo de dependências pelos construtores. Uso `fx.Annotate` para fornecer o adaptador PostgreSQL pela interface do domínio. O `fx.Invoke` força a construção do servidor. Nos hooks, verifico o banco antes de abrir HTTP e encerro o HTTP antes de fechar o pool.”

Pontos que vale explicar:

- `fx.Provide`: registra como construir uma dependência; não significa que tudo é instanciado imediatamente.
- `fx.Lifecycle`: inicializa na ordem de registro e encerra na ordem inversa.
- Porta ocupada: startup falha e o pool iniciado é fechado.
- Ping falha: fecho explicitamente o pool porque o hook que falhou não recebe `OnStop`.
- Domínio e adaptador PostgreSQL não importam Fx.

## 3:50–4:40 — Testes e entrega

Abra [GitHub Actions](https://github.com/yubr02/lucky-seven/actions) e mostre uma execução concluída com sucesso.

> “Tenho testes das combinações, validação HTTP e integração com PostgreSQL real. Testo várias requisições com a mesma chave, gasto concorrente, saldo insuficiente e rollback. O CI executa race detector, build do frontend e um teste HTTP com Docker, inclusive após reiniciar os containers.”

Distinga as validações: os testes Go rodam no CI; os testes de duas abas e aparência foram exercitados localmente no Chrome e ainda não são uma suíte de navegador integrada ao CI.

## 4:40–5:30 — Limitações e próximos passos

> “É uma demonstração com carteira única e sem autenticação. O hub WebSocket é em memória e atende uma instância. Pode haver perda de aviso entre commit e publicação; por isso a reconexão consulta o banco. Para uma entrega confiável e distribuída, eu estudaria outbox e mensageria. Também quero separar melhor a coordenação transacional do caso de uso, que hoje está no adaptador PostgreSQL.”

Não afirme ter implementado AWS, arquitetura de microsserviços ou DDD completo. A vaga valoriza base consistente e aprendizado; deixar claros os limites faz parte da explicação.

## Perguntas para treinar

**Por que `crypto/rand`?**
Para obter números imprevisíveis com uma fonte criptográfica. `rand.Int` com limite 10 evita usar módulo de um valor aleatório, que pode introduzir viés. Isso sozinho não torna o jogo auditado ou certificado.

**E se a resposta HTTP se perder depois do commit?**
O cliente guarda chave e aposta e tenta recuperar usando a mesma chave. O banco devolve o resultado existente, sem outro sorteio ou débito.

**Por que a transação não basta para concorrência?**
Ela agrupa alterações atomicamente, mas precisamos controlar a leitura/modificação concorrente do saldo. O bloqueio da linha da carteira serializa esse fluxo.

**Por que não enviar o saldo diretamente no WebSocket?**
O aviso serve para invalidar a visão local. Buscar o estado atual evita aplicar um saldo antigo quando mensagens chegam repetidas ou fora de ordem. Há custo adicional de consultas REST.

**O que acontece quando o WebSocket cai?**
O frontend tenta reconectar com espera progressiva de 1 a 15 segundos. Ao receber `ready`, consulta saldo e histórico. As apostas continuam por REST, se a API estiver acessível.

**Como o cliente lento é tratado?**
Uma notificação pendente por conexão é suficiente, pois ela pede nova consulta. O publicador não espera I/O; escritas têm prazo, e heartbeat detecta conexões sem resposta.

**Você usou IA?**
> “Sim, usei assistência de IA para implementar e investigar. Minha responsabilidade é revisar, testar e explicar as decisões. Posso mostrar as validações e os pontos que ainda preciso aprofundar.”

**O que você mudaria para produção?**
Identidade autenticada e autorização por carteira, auditoria de movimentações, limites de uso, observabilidade, migrations versionadas, permissões do banco, gestão de segredos e estratégia de eventos duráveis conforme os requisitos.

**O que você sabe de AWS?**
Explique seu conhecimento real. Este projeto ainda não usa serviços AWS. Como estudo futuro, SQS poderia transportar eventos para um consumidor separado, com tratamento de duplicação e falhas.

## Encerramento

> “Esse projeto me ajudou a acompanhar uma requisição do HTTP até o banco e a validar falhas e concorrência. Quero continuar evoluindo com revisão e orientação, principalmente na organização dos casos de uso e nos fluxos assíncronos.”
