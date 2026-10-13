# Apresentação nativa nesta máquina

Não há deploy público. A URL do jogo é local: `http://localhost:5173`. A API desta máquina usa a porta `8081`, e o PostgreSQL, `5432`.

## Se os serviços já estiverem ligados

Abra duas abas em `http://localhost:5173`. Confirme “Atualizações em tempo real conectadas”. Clique em Atualizar no histórico para conferir os dados.

Verificação da API no PowerShell:

```powershell
Invoke-RestMethod http://localhost:8081/health
Invoke-RestMethod http://localhost:8081/api/balance
```

Não inicie outra cópia da API na mesma porta. O GitHub contém o código; o link `localhost` funciona somente na máquina que está executando os serviços.

## Iniciar após reiniciar o Windows

Os comandos partem de `C:\Users\pitoco\Jungle`. O `.env` local e os binários em `.local/pgsql` já foram preparados nesta máquina e não estão no Git.

### 1. PostgreSQL

```powershell
cd C:\Users\pitoco\Jungle
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/start-local-db.ps1
```

O helper mantém os dados existentes. Não recrie o banco ou execute novamente a migração manualmente para preparar a apresentação.

### 2. API — primeiro terminal

```powershell
cd C:\Users\pitoco\Jungle
$dbSetting = Get-Content .env | Where-Object { $_ -match '^DATABASE_URL=' }
if (-not $dbSetting) { throw 'Configure DATABASE_URL no .env.' }
$env:DATABASE_URL = $dbSetting.Substring(13)
$env:API_PORT = '8081'
cd backend
go run ./cmd/api
```

Mantenha o terminal aberto. `Ctrl+C` solicita o encerramento pelo Fx. Se aparecer porta ocupada, confira a API que já está rodando antes de iniciar outro processo. Não exiba a URL de conexão com senha durante a entrevista.

### 3. Frontend — segundo terminal

Confira que o `.env` da raiz contém `VITE_API_TARGET=http://localhost:8081`.

```powershell
cd C:\Users\pitoco\Jungle\frontend
npm.cmd run dev -- --port 5173 --strictPort
```

Abra `http://localhost:5173`. Se precisar instalar dependências, execute `npm.cmd ci` antes da apresentação, não durante a chamada.

## Validar antes de apresentar

Em um terceiro terminal:

```powershell
cd C:\Users\pitoco\Jungle
$dbSetting = Get-Content .env | Where-Object { $_ -match '^DATABASE_URL=' }
$env:TEST_DATABASE_URL = $dbSetting.Substring(13)
$env:REQUIRE_DATABASE_TESTS = '1'
cd backend
go test -count=1 ./...
go vet ./...
```

Os testes de integração usam schemas temporários separados, sem gastar créditos da carteira demo. Faça uma jogada de 1 crédito pela interface para verificar o fluxo visual e a segunda aba.

## Plano B

- Sem internet: o jogo e o banco locais continuam funcionando, desde que as dependências já estejam instaladas.
- Sorteio sem prêmio: use o laboratório de demonstração para mostrar par, trio ou 777, deixando explícito que são exemplos.
- Saldo esgotado: use o laboratório para demonstrar as combinações e explique a rejeição de saldo insuficiente. Confira o saldo antes da chamada.
- WebSocket desconectado: o botão Atualizar ainda consulta saldo e histórico por REST, se a API estiver disponível.

Use o [roteiro de apresentação](roteiro-apresentacao.md) para organizar a explicação e preparar perguntas técnicas.
