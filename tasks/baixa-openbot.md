# Baixa Collecta — agente do OpenBot

Decisões do dono (2026-11-03): MCP travado, agente separado do atendente, ao vivo direto.
Contexto: o launchd `com.collecta.baixa-vendas` parou em 28/10 (`exit 78`, `Operation not permitted`, TCC do macOS na pasta Documents).

## Desenho

- Agente "Baixa Collecta" no OpenBot, provedor Claude Code, perfil de navegador próprio logado na Liga (`www.ligapokemon.com.br`, host do admin) e no MyP.
- Navegador e `run_script` liberados só em `ligamagic.com.br`, `ligacollectatcg.com.br`, `ligaonepiece/ligapokemon/ligariftbound.com.br`, `mypcards.com`.
- Sem terminal, arquivos, web, skills ou subagentes. O admin token fica só no MCP.
- Rotina do OpenBot: 9h, 14h e 20h, com o texto de `bot/baixa/rotina.md`.

## MCP `cmd/baixa-mcp` (servidor `collecta-baixa`)

Env: `OPDEALS_URL`, `OPDEALS_ADMIN_TOKEN`, `NTFY_TOPIC`, `BAIXA_DADOS` (padrão `~/.collecta-baixa`).

Ferramentas:

- [x] `pedidos_baixados(canal)`: números de pedido `Liga #N` / `MyP #N` já no ledger (via `/api/sales`), para o dedup.
- [x] `estoque_buscar(jogo, numero?, nome?)`: holdings com id, nome, número, variante, condição, qty, askBRL, flag Na Liga.
- [x] `dar_baixa(jogo, trade_id, qty, preco_brl, canal, pedido, comprador, data)`: monta o `buyer` no formato da skill e chama `/api/trades/{id}/sell` com `channel`. Recusa se aquele pedido já baixou aquele registro, se qty passa do disponível ou se o canal não é `liga`/`myp`. Relê o registro e devolve o estado.
- [x] `fila_sincronizar()`, `fila_feita(id)`, `fila_erro(id, motivo)`: `/api/syncqueue`.
- [x] `posvenda_liga(pedidos)`: `POST /api/postsale/liga-orders` (Fase 7 da `liga-vendas`). Só primeiro nome, telefone, itens e retirada.
- [x] `relatorio(resumo, corpo)`: grava `<BAIXA_DADOS>/<carimbo>.md` e manda o resumo pro ntfy. Recusa CPF e endereço no corpo.
- [x] Testes no estilo `cmd/atendente-mcp/main_test.go` (httptest fake do opdeals): dedup, qty acima do disponível, canal inválido, relatório.

Fora da v1 (vira pendência no relatório, como no `--auto`): lote extra quando a Liga está acima do dashboard e redução do dashboard.

## Instruções do agente `cmd/baixa-mcp/instructions.md`

- [x] Ler inteiras `baixa-vendas`, `liga-vendas` e `myp-vendas` e portar para o agente: URLs, leitura dos pedidos, casamento por nome + número + variante, propagação Liga↔MyP, fila Sincronizar, gêmeos JP, selados (`prod/all`), `requestSubmit`, 8 s entre páginas, parar no Cloudflare.
- [x] Regras do `--auto`: nunca pergunta nem chuta, ambíguo vira pendência, venda dupla `URGENTE`.
- [x] Trocar curl/arquivo pelas ferramentas do MCP.

## Trava `bot/baixa/`

- [x] `settings.json` (mesmo deny list do atendente).
- [x] `guard.sh`: navegação permitida só nos domínios acima; `run_script` permitido; upload/download/senha bloqueados; log em `~/.collecta-baixa/bloqueios.log`.
- [x] `ferramentas-permitidas.txt`: `mcp__collecta-baixa__.*`, ferramentas de busca do OpenBot.
- [x] `rotina.md`, `install.sh` (instala em `/Users/Shared/collecta-bot/baixa/`), alvo `make baixa`, `SETUP.md` passo a passo.

## Verificação

- [x] `go test ./cmd/baixa-mcp/` e `go vet`.
- [x] `sh -n` no `guard.sh` e testes do guard com JSON de entrada (domínio permitido, domínio fora, upload).
- [x] Rodada assistida: você cria o agente, eu acompanho a 1ª rodada no chat do OpenBot e confiro cada baixa no ledger de prod.
- [ ] Rotina 9h/14h/20h criada no OpenBot.

## Revisão

2026-11-03: `cmd/baixa-mcp` com 10 ferramentas (o `pedidos_baixados` virou parte do `inicio`), 11 testes passando, `go vet` limpo. `instructions.md` com 401 linhas, portado das 3 skills. `bot/baixa/` com trava, guard testado com 13 entradas (domínio falso, userinfo, http, upload e Bash bloqueados), `make baixa` instalado em `/Users/Shared/collecta-bot/baixa`. Fumaça só de leitura na prod: `inicio` com 79 pedidos Liga e 1 MyP, `estoque_buscar` e `fila_sincronizar` ok.
Decisões do port: pedidos da Liga anteriores a 07/10/2026 13:11 são ignorados; qty acima do estoque vira `venda_sem_lastro` do resto; `restore` sem anúncio no MyP vira pendência; modelo `opus` no settings.json.
Falta: criar o agente no OpenBot, logar o navegador, 1ª rodada assistida e agendamento (SETUP.md passos 2 a 9).

2026-11-03 noite: agente "Estoquista Collecta" no ar. 1ª rodada (19:48) baixou o Mr.3 OP16-056 (4 un, R$ 156,24, Liga #12385475) e deixou Brook OP15-032 e Boa Hancock OP16-032 como pendência por lotes idênticos; os dois foram baixados à mão e a regra virou "lote com mais unidades". Binário reinstalado com 11 ferramentas (`registrar_pendencia`, `pendencias_abertas` no `inicio`).
2ª rodada (20:05, regras 3efa51b5): nada novo, nenhuma gravação; ledger dos 3 jogos idêntico à foto de antes (380/531/368 registros), fila Sincronizar vazia, pós-venda 0 criados e 4 ignorados. O MCP recusou um relatório com telefone e o agente reenviou sem. Achou 3 pedidos Pokémon cancelados depois de pagos que têm baixa no dashboard: #12373980 (Fennekin 085, Braixen TG01), #12356004 (Wailord ex 016, Mega Chandelure ex 038, Goldeen 084) e #12327804 (Parasect TG01, vendido a R$ 0,001). Ficam para o dono decidir; não há ferramenta para desfazer venda.
Descobertas: o painel MCP do OpenBot é global (não existe MCP por agente), então a isolação depende só das travas; não há restart de MCP, cada conversa abre um processo novo. O agente usou `ligapokemon.com.br` porque `ligamagic.com.br` estava deslogado. Cron `com.collecta.baixa-vendas` descarregado.
Rotina "Baixa de vendas" criada no Estoquista pelo computer-use: cron `0 9,14,20 * * *`, America/Sao_Paulo, ativa, 1ª execução 04/11 9h. "Manter acordado" já estava ligado no app (`keepAwake: true`, mas `keepAwakeOnBattery: false`: só roda com o Mac na tomada).
Bug achado na 2ª rodada: o Claude Code corta as instruções de servidor MCP em ~4 KB e o agente só viu as regras 1 a 7 (de 26 KB). Correção: o servidor manda só um aviso curto (336 bytes) e o `inicio` devolve as regras inteiras no campo `regras`. Teste novo `TestServerInstructionsFitClientLimit`, `go test` ok, `make baixa` instalado. O atendente (`cmd/atendente-mcp/instructions.md`, 26,5 KB) tem o mesmo corte.
2026-11-03 23h: as rodadas das 20:47 e 23:07 não leram a Liga. A regra de hosts mandava `www.ligamagic.com.br` e depois `www.ligacollectatcg.com.br` e dizia que o domínio do jogo não servia o admin; a sessão do dono está em `www.ligapokemon.com.br` (a Liga guarda a sessão por host), e a rodada das 19:48 só funcionou porque o agente desobedeceu a regra. Correção: admin da loja é `https://www.ligapokemon.com.br/?view=ecom/admin/home`, hosts de reserva ligamagic e ligacollectatcg, sessão caída só com os três deslogados, URL de gravação de `cartas/all` também no host da Pokémon. `go test` ok, `make baixa` reinstalado (`versao_regras` muda sozinha pelo hash). SETUP passo 5 aponta para o host certo. Conferir na rodada das 9h de 04/11 se a listagem foi lida.
Rodada 23:15 (regras f91f91bc, host www.ligapokemon.com.br logado): 6 baixas (#12387121 Gardevoir TG05, Eevee TG11, Pikachu 027 SVP; #12386159 Base Shop Vol.1, Dewgong 084 JP, Medicham 207 JP), conferidas na prod com `canal liga`, comprador `Liga #N` e `liga_qty` descontado (Pikachu 2→1, Dewgong 4→3, Medicham 5→4). Venda sem lastro #12387115 (Caixa 151 Surprise CHN, R$ 310) gravada com `semLastro`. Pós-venda 3 criados. Erro do agente no relatório: disse que #12373980 "não estava baixado", mas Fennekin 085 e Braixen TG01 estão vendidos com `Liga #12373980` em 02/11; são 3 cancelados pós-baixa, não 2.
