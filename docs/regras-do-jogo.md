# Regras do jogo

Regras acordadas em 13/10/2026. Créditos fictícios; saldo inicial de 1.000.

- Três números independentes e uniformes de 0 a 9, com repetição, sorteados por `crypto/rand` no backend.
- Aposta inteira de 1 a 100 créditos. O jogador escolhe apenas o valor.
- `777`: retorno de 100×; outros três iguais: 20×; exatamente dois iguais em qualquer posição: 2×; todos diferentes: zero.
- Prêmios não se acumulam. O retorno inclui a aposta: `saldo final = saldo anterior - aposta + retorno`.
- Saldo insuficiente: rejeitar antes do sorteio, sem débito nem entrada no histórico de jogadas.
- Débito, crédito do prêmio e registro do resultado são confirmados na mesma transação PostgreSQL. Falhas desfazem todas as alterações.

## Contrato HTTP

A demonstração usa a carteira fixa `demo`, provisionada pela migração. Não há autenticação ou seleção de jogadores; este contrato é para demonstração local.

- `POST /api/play`: corpo `{"bet":10}` e header obrigatório `Idempotency-Key` (1–128 caracteres ASCII: letras, números, `_` ou `-`). Resposta 200 com `id`, `numbers`, `bet`, `payout`, `balance` e `created_at`.
- `GET /api/balance`: saldo atual em `{"balance":1000}`.
- `GET /api/history`: até 50 resultados, mais recentes primeiro. Próxima página: `?before=<menor id recebido>`. Histórico vazio retorna `[]`.
- Erros: 400 `invalid_request`, 422 `insufficient_balance`, 409 `idempotency_conflict`, 500 `internal_error`.

A chave é única por jogador. Repetir chave e aposta devolve o resultado persistido, incluindo o saldo na ocasião, sem novo sorteio ou cobrança. Outra aposta com a mesma chave retorna conflito. A consulta da chave precede a validação de saldo para permitir repetir uma jogada mesmo após gastar o saldo restante.

Requisições rejeitadas ou transações revertidas não reservam a chave. Após falha de rede ou resposta incerta, reenviar a mesma chave e aposta. Chaves concluídas são mantidas junto ao histórico, sem expiração. O bloqueio da carteira serializa apostas concorrentes; uma restrição única no banco também impede duplicação.
