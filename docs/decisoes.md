# Decisões de arquitetura

- Monorepo: API Go em `backend/`, interface React em `frontend/` e documentação em `docs/`.
- API com `net/http`; configuração e timeouts por ambiente; encerramento gracioso.
- `GET /health` verifica apenas a vida da aplicação. A inicialização exige conexão PostgreSQL via `DATABASE_URL`.
- `internal/game` contém validação, sorteio com `crypto/rand` e premiação. O serviço depende de uma interface de persistência.
- `internal/storage/postgres` usa pgx e transação READ COMMITTED. `SELECT ... FOR UPDATE` bloqueia a carteira antes de consultar a idempotência e o saldo; requisições concorrentes veem o resultado confirmado pela anterior.
- Débito, prêmio e histórico são atômicos. A resposta persistida permite recuperar o resultado caso a conexão HTTP falhe após o commit.
- A migração SQL é aplicada explicitamente uma vez, antes da API, e provisiona a carteira local `demo` com 1.000 créditos. Reiniciar a API não repõe créditos.
- O contrato atual usa um único jogador de demonstração, sem autenticação. A identidade deverá vir de autenticação antes de disponibilizar múltiplas carteiras publicamente.
- O frontend consome a API via proxy Vite e oferece aposta, saldo, resultado e histórico paginado. A tentativa pendente é persistida em localStorage antes do envio e reutilizada em falhas de rede para evitar cobrança duplicada.
- Testes de integração usam `TEST_DATABASE_URL` e criam/removem um schema isolado por teste, sem limpar tabelas existentes.
