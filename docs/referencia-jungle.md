# Referência de produto: Jungle Gaming

Análise do site público https://junglegaming.io realizada em 13/10/2026.

## O que a empresa apresenta

A Jungle Gaming se apresenta como fornecedora B2B de jogos originais, agregação de jogos, plataforma operacional modular (Turnkey) e inteligência de jogadores com IA. O site destaca integração entre jogos, carteira, operações e análise. Isso descreve a oferta comercial publicada, não uma validação independente dos serviços.

## Aplicação neste projeto

- Direção visual inspirada no site: fundo quase preto, tipografia de grande escala, botões arredondados, cartões de jogo em destaque e seções espaçadas.
- Arte própria em HTML/CSS para o Lucky Seven, com tons verdes e dourados; nenhuma arte dos jogos comerciais foi incorporada.
- Jornada integrada: apresentação do jogo, acesso à aposta, carteira, regras, resumo e histórico.
- Resumo calculado a partir das jogadas carregadas: quantidade, soma das apostas e soma dos retornos. A amostra é explicitada e muda ao carregar páginas anteriores.
- Navegação por âncoras, layout responsivo, foco de teclado e respeito à preferência por movimento reduzido.
- Fluxo de jogada, premiação, PostgreSQL e idempotência preservados.

O projeto continua sendo uma demonstração independente de portfólio, com um jogo e créditos fictícios. Não implementa o catálogo, pagamentos, KYC, CRM, IA ou serviços comerciais da Jungle Gaming. O rodapé identifica a inspiração sem apresentar a demonstração como um produto oficial.

## Validação

Build TypeScript/Vite e teste no Chrome: aposta inválida bloqueada, jogada de 1 crédito, recuperação de resposta perdida após recarregar sem duplicação, atualização do histórico e layout em 390 px.
