# Lucky Seven

Jogo de portfólio com backend **Go**, **PostgreSQL** e interface **React/TypeScript**. O foco é mostrar um fluxo completo de aposta fictícia: validar, sortear, atualizar créditos e registrar o resultado de forma atômica, com proteção contra cobrança duplicada.

## O que está implementado

- API REST com `net/http`, validação de entrada e erros de domínio; composição e ciclo de vida com Uber Fx.
- Sorteio uniforme com `crypto/rand` e regras de premiação testadas nas 1.000 combinações possíveis.
- Transação PostgreSQL com bloqueio da carteira e chave de idempotência por jogador.
- Histórico paginado, saldo persistente e recuperação de resposta perdida.
- Testes de integração com concorrência, saldo insuficiente e rollback.
- Interface responsiva com animação, som opcional, detalhes das jogadas e laboratório de exemplos isolado da API.
- Dockerfile em duas etapas e workflow de GitHub Actions para testes, build e teste HTTP dos containers.

## Executar com Docker

Requisitos: Docker com Compose v2. Para a interface, Node.js 22 e npm.

Na raiz, crie `.env` a partir de `.env.example` **se ainda não existir** e preencha `POSTGRES_PASSWORD` com uma senha de desenvolvimento:

```powershell
Copy-Item .env.example .env
notepad .env
docker compose up --build -d --wait
```

A API fica em `http://localhost:8080`. O banco é inicializado automaticamente **somente quando o volume está vazio**, incluindo a carteira `demo` com 1.000 créditos. Reiniciar os serviços mantém o saldo.

O Compose configura a conexão com variáveis `PGHOST`, `PGDATABASE`, `PGUSER` e `PGPASSWORD`. Não usa a `DATABASE_URL` do arquivo local, cujo endereço pode apontar para outro banco.

Se já houver PostgreSQL ou API nas portas padrão, ajuste `POSTGRES_PORT=5433` e `API_PORT=8082` no `.env`. Para conectar a interface à API escolhida, ajuste também `VITE_API_TARGET=http://localhost:8082`.

```powershell
docker compose ps
docker compose logs -f api
docker compose down
```

`down` preserva o volume. Não use `down -v` se quiser manter seus dados.

### Banco existente sem tabelas

O script de inicialização do PostgreSQL não é reexecutado em volumes existentes. Se o banco já existia antes desta versão e ainda não possui as tabelas, aplique a migração uma única vez:

```powershell
Get-Content -Raw backend/migrations/001_game.sql | docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

A migração atual é um bootstrap explícito; ainda não há controle de versões de migrações para futuras alterações.

## Interface

```powershell
cd frontend
npm ci
npm run dev
```

Abra `http://localhost:5173`. O Vite encaminha `/api` e `/health` para `VITE_API_TARGET`, definido no `.env` da raiz. Reinicie o Vite ao alterar o destino. O frontend ainda roda separadamente dos containers.

O histórico exibe dez registros por página; os filtros atuam sobre as jogadas carregadas, com essa quantidade informada na tela. Detalhes mostram aposta, retorno e saldo antes/depois. O laboratório apresenta resultados e erros simulados sem acessar a API nem alterar a carteira.

## Backend sem Docker

Requisitos: Go 1.24+ e PostgreSQL 17 com a migração aplicada.

```powershell
cd backend
$env:DATABASE_URL = 'postgres://lucky_seven:SUA_SENHA@localhost:5432/lucky_seven?sslmode=disable'
go mod download
go run ./cmd/api
```

O Go não carrega `.env` automaticamente. Codifique caracteres especiais da senha para URL. `sslmode=disable` é apenas para desenvolvimento local. Como alternativa à URL, o driver aceita as variáveis PostgreSQL `PG*`.

O helper `scripts/start-local-db.ps1` atende ao workspace Windows já preparado, com binários PostgreSQL em `.local/pgsql`. Não é requisito para usar Docker e não baixa os binários automaticamente.

## Testar a API

```powershell
Invoke-RestMethod http://localhost:8080/api/balance
$key = [guid]::NewGuid().ToString()
Invoke-RestMethod http://localhost:8080/api/play -Method Post -ContentType 'application/json' -Headers @{ 'Idempotency-Key' = $key } -Body '{"bet":10}'
Invoke-RestMethod http://localhost:8080/api/history
```

Repetir chave e aposta devolve o resultado original. Alterar a aposta com a mesma chave retorna 409. Gere outra chave para uma nova jogada.

| Endpoint | Comportamento |
|---|---|
| `GET /health` | Liveness; não verifica banco ou migração |
| `POST /api/play` | Valida aposta, sorteia e confirma resultado e saldo |
| `GET /api/balance` | Saldo atual |
| `GET /api/history?before=ID` | Até 50 resultados anteriores ao cursor |

## Arquitetura e fluxo

```mermaid
sequenceDiagram
    participant UI as React
    participant HTTP as Handler HTTP
    participant Game as Serviço de jogo
    participant Store as PostgreSQL repository
    participant DB as PostgreSQL
    UI->>HTTP: POST /api/play + Idempotency-Key
    HTTP->>Game: Validar aposta e chave
    Game->>Store: Executar jogada
    Store->>DB: BEGIN + bloquear carteira
    Store->>DB: Consultar resultado da chave
    alt Resultado já existe
        Store-->>UI: Resultado persistido, sem novo débito
    else Nova chave e saldo suficiente
        Store->>Store: crypto/rand + cálculo do prêmio
        Store->>DB: Atualizar saldo + inserir resultado
        Store->>DB: COMMIT
        Store-->>UI: Resultado confirmado
    end
```

O serviço define regras e contrato de persistência. O adaptador PostgreSQL coordena a transação e chama o sorteio somente após verificar chave e saldo. Essa coordenação ainda está no adaptador: uma próxima evolução é explicitar melhor a unidade de trabalho no caso de uso.

Veja [regras e contrato](docs/regras-do-jogo.md), [decisões](docs/decisoes.md) e [referência visual](docs/referencia-jungle.md).

## Testes e CI

```powershell
cd backend
$env:TEST_DATABASE_URL = $env:DATABASE_URL
$env:REQUIRE_DATABASE_TESTS = '1'
go test -count=1 ./...
go vet ./...
```

Os testes PostgreSQL criam schemas isolados e precisam de permissão para criá-los. Sem `TEST_DATABASE_URL`, normalmente são pulados; com `REQUIRE_DATABASE_TESTS=1`, sua ausência causa falha para evitar CI falsamente verde.

O workflow `.github/workflows/ci.yml` executa em push, pull request ou manualmente:

1. Formatação, `go vet` e testes com `-race`, usando PostgreSQL real.
2. Instalação pelo lockfile e build TypeScript/Vite.
3. Build da imagem, inicialização do Compose e teste HTTP de aposta/idempotência.
4. Reinício dos containers e repetição do teste para verificar persistência.

`scripts/smoke-api.py` usa uma chave fixa e pressupõe uma carteira descartável iniciada com 1.000 créditos. Execute somente no banco temporário de CI, nunca na carteira usada para jogar.

O CI roda no [GitHub Actions](https://github.com/yubr02/lucky-seven/actions). Ele valida também os containers em Linux; os testes nativos não substituem esse check.

## Limitações e próximos passos

- Uma carteira fixa de demonstração, sem autenticação ou dinheiro real.
- PostgreSQL local usa o usuário de desenvolvimento; produção exigiria separação de permissões e gestão de segredos.
- Ainda não há AWS ou divisão em microsserviços. As notificações WebSocket atendem a uma única instância da API.
- Evoluir em mudanças pequenas: evolução do caso de uso e um fluxo assíncrono delimitado.

O código foi desenvolvido com assistência de IA. As decisões, os testes e as limitações estão documentados para permitir revisão e explicação pelo autor.

## Uber Fx

`cmd/api/main.go` carrega a configuração e executa `app.New(cfg).Run()`. O módulo em `internal/app` registra os construtores do pool PostgreSQL, repository, serviço de jogo, rotas e servidor HTTP. `fx.Annotate` expõe o adaptador PostgreSQL como `game.Repository`, mantendo o domínio independente do Fx.

O hook do banco faz `Ping` antes de o HTTP abrir a porta. A abertura da porta é síncrona: se estiver ocupada, o startup falha e o Fx fecha o pool já iniciado. No encerramento, os hooks rodam na ordem inversa: HTTP conclui requisições em andamento, depois o pool é fechado. Se o prazo HTTP expirar, as conexões são encerradas. Uma falha inesperada no servidor solicita shutdown com código de saída 1.

Os testes validam o grafo de dependências, a inicialização e parada reais, falha de conexão e rollback após erro de bind. O pool é fechado explicitamente se o próprio hook de inicialização falhar, pois nesse caso o Fx não executa seu `OnStop`.

Referência: [ciclo de vida oficial do Uber Fx](https://uber-go.github.io/fx/lifecycle.html).
## Atualizações em tempo real

`GET /api/events` abre um WebSocket de notificações da carteira fixa `demo`. A aposta continua em `POST /api/play`. Somente depois de o serviço retornar sucesso (commit ou replay persistido), o handler publica `{"type":"wallet.changed"}`. Falhas de validação, saldo ou persistência não publicam eventos.

Ao conectar, o cliente recebe `{"type":"ready"}` e consulta saldo e histórico por REST. O frontend repete essa consulta após notificações, ao voltar à aba e ao reconectar. A reconexão usa espera progressiva de 1 a 15 segundos; atualizações ficam adiadas durante uma jogada/animação local. Notificações não alteram o resultado da máquina nem geram apostas.

O hub é em memória e envia apenas avisos, sem saldo no payload. Há até 64 clientes, uma notificação pendente por cliente, timeout de escrita, heartbeat e encerramento pelo ciclo de vida Fx. O proxy Vite preserva Host no WebSocket para manter a verificação de Origin, sem liberar origens arbitrárias.

Limitações: entrega best-effort, sem fila durável ou garantia de entrega entre commit e publicação. Um replay pode gerar outro aviso de atualização, sem cobrar novamente. Escritas diretas no banco não notificam. Múltiplas instâncias precisariam de distribuição de eventos; o banco permanece a fonte de verdade. A demo não tem autenticação, portanto todos os clientes observam a mesma carteira fictícia.

Validação: testes Go verificam duas conexões, notificações somente em sucesso, coalescência concorrente, nova conexão, shutdown e rejeição de origem externa. O fluxo de duas abas e recuperação após ficar offline também foi exercitado no Chrome com a API local.