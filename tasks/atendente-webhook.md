# Atendente acordado por evento, não por cron (plano)

Base: pesquisa de 04/11/2026 no OpenBot 0.60.0 (bundle extraído), no `openbot.db` (só leitura) e no servidor do pós-venda.

## Situação hoje

- Rotina "Atendimento wpp" no cron `*/5 8-21 * * *`: até 168 rodadas por dia. Em 03/11 rodaram 45 (o Mac dormiu parte do dia) e 33 terminaram sem nada (`[[no-update]]`).
- Desde 04/11 a rodada vazia lê 7,8 KB de regras em vez de 28,6 KB (modo enxuto). O que sobra de custo fixo é a sessão do OpenBot e a abertura do WhatsApp e do Instagram.

## O que a pesquisa achou

1. **OpenBot tem duas formas de disparar uma rotina de fora:**
   - **Webhook de rotina:** passa pela nuvem (`https://signal.openbot.run/v1/webhooks/<rota>`), exige conta logada e relay conectado, assinatura HMAC com segredo `whsec_…`. Funciona, mas é mais peça para quebrar.
   - **Local scripts (escolhido):** API só em loopback, ligada pela chave "Local scripts" nas configurações do agente. Ao ligar, o OpenBot grava `~/Library/Application Support/OpenBot/automation/{url,token,headers}` (porta e token mudam a cada abertura do app). `POST <url>/v1/agents/<agentId>/routines/<routineId>/run` com `Authorization: Bearer <token>` e corpo `{"payload": "..."}` (até 4.000 caracteres) responde 202. O payload chega ao agente como "evento de script local". Limite: 30 chamadas por hora por agente. Sem dedupe: cada chamada é uma rodada nova na fila do agente.
   - Pelo código, a chamada por script roda mesmo com a rotina pausada (o agendador só dispara rotina ativa; o run por script não confere). Não testado.
   - Agente: `agent-d9a81719-259c-4cb0-b60b-c1bdec9ef8df`. Rotina: `76cadb3e-90c6-4557-a014-d683a0f5d7ac`.
2. **Não existe sinal limpo, sem LLM, de mensagem não lida no WhatsApp Web ou no Instagram dentro do OpenBot.** O navegador não expõe porta de DevTools, não guarda o título da aba, não pode mandar notificação do macOS, e o WhatsApp Desktop não está instalado. O IndexedDB do WhatsApp Web está em disco, mas é LevelDB travado pelo app, com formato que muda sem aviso: fora deste plano.
3. **Pós-venda dá para saber pelo servidor.** O servidor (Railway) já sabe quando há primeira mensagem para escrever (`compose`), mensagem na fila (`outbox` não reservado), rastreio a avisar e conversa aberta esperando resposta do cliente.
   - Cuidado: toda chamada às rotas `/api/postsale/agent/*` atualiza `agentSeen`, que alimenta o alerta "OpenBot parado". Um vigia que consultasse essas rotas esconderia o alerta. `outbox/claim` ainda por cima reserva as mensagens.
   - O alerta de parado dispara com 15 min sem o agente. Com rodadas menos frequentes ele vira alarme falso.

## Desenho

Um vigia local (`cmd/atendente-vigia`, Go, launchd) passa a ser o único dono da cadência. O cron do OpenBot sai.

A cada 60 s, dentro do horário (8h às 22h):

1. Pergunta ao servidor o que há de pós-venda em `GET /api/postsale/agent/pending` (token do agente, sem mexer no `agentSeen`).
2. Decide se acorda o agente:
   - **Pós-venda pendente** (compose, outbox, rastreio): acorda na hora.
   - **Conversa de pós-venda aberta** (cliente pode responder a qualquer momento): acorda se a última rodada foi há mais de 5 min.
   - **Nada pendente:** acorda se a última rodada foi há mais de 20 min (batimento para WhatsApp e Instagram orgânicos).
   - Nunca acorda se acordou há menos de 3 min (no máximo 20 por hora, abaixo do limite de 30 do OpenBot).
3. Para acordar: lê `automation/url` e `automation/token` na hora (mudam a cada abertura do OpenBot) e chama o run da rotina com um payload curto, por exemplo `pós-venda: 2 primeiras mensagens, 1 envio` ou `batimento`.
4. Registra cada decisão em `~/.collecta-atendente/vigia.log` (hora, motivo, resposta do OpenBot). Esse arquivo também resolve o rastro que falta hoje das rodadas vazias.

Se o OpenBot estiver fechado (sem a pasta `automation`) ou responder erro, o vigia anota e tenta no próximo minuto; depois de 15 min falhando, avisa pelo ntfy.

### Rodadas estimadas por dia

- Batimento de 20 min por 14 h: cerca de 42.
- Mais as rodadas puxadas por pós-venda, de 5 em 5 min só enquanto houver conversa aberta (janela de 12 h depois da oferta).
- Contra 168 do cron atual. Os números reais vêm da fase 4.

## Fases

### Fase 0: validar o OpenBot (feita em 04/11/2026)
- [x] "Local scripts" ligado. Fica em Agents → Local scripts (tela própria, não nas configurações do agente). O Default veio ligado para todos os agentes; o certo é Default desligado e só o Atendente em "On".
- [x] Pasta `automation` apareceu (`url` = `http://127.0.0.1:<porta>`, `token`, `headers`). Disparo com payload `teste do vigia` respondeu 202 com `runId` e `deliveryId`; a rodada entrou como `run_kind=manual`, `run_source=script` e terminou `succeeded` em 29 s.
- [x] Com a rotina pausada (`active=0`), o disparo por script rodou e terminou `succeeded` em 48 s. Plano B descartado: o cron pode ficar pausado.
- Achado útil para o vigia: o estado de cada rodada está em `projection_routine_runs` do `openbot.db` (`status` queued/running/succeeded/failed…, `run_source`, `created_at`). O vigia pode ler isso só leitura para saber se ainda há rodada na fila ou rodando antes de disparar outra, e quando foi a última rodada de verdade.

### Fase 1: servidor (código pronto em 04/11/2026, falta deploy)
- [x] `GET /api/postsale/agent/pending` (`internal/api/postsale_pending.go`). Mudou do plano: usa o token do agente (`POSTSALE_AGENT_TOKEN`), não o de admin, porque ele só lê o pós-venda e não mexe em estoque; passa por `agentPeek`, que confere o token sem atualizar o `agentSeen`. Só existe no modo OpenBot (`POSTSALE_TRANSPORT=openbot`).
  - Resposta: `{"quiet", "compose", "outbox", "tracking", "open", "oldestAt"}`. `compose`, `outbox` e `tracking` usam as mesmas regras do `claim` e do `trackingDue` (item reservado há menos de 10 min não conta; resposta com mais de 2 h não conta). `open` são as conversas esperando o cliente, com janela aberta. `oldestAt` diz desde quando o item mais antigo espera.
  - No horário silencioso (22h às 8h), primeira mensagem, oferta e rastreio não contam, e o que entrou à noite conta como esperando desde as 8h.
- [x] Alerta de "OpenBot parado": mudou do plano. Ficou mais simples que "45 min": dispara quando há item esperando há mais de 15 min **e** o agente não apareceu desde que ele chegou (referência = o mais recente entre `agentSeen` e `oldestAt`). Agente parado entre batimentos não dispara mais nada, e conversa aberta sozinha também não. Saiu o `outboxSince`; a mensagem do ntfy diz há quantos minutos o item espera e lembra de olhar o vigia.
- [x] Testes: `TestOpenBotStaleAgentAlertDecision` (reescrito), `TestOpenBotStaleAlertIgnoresIdleAgentBeforeWork`, `TestOpenBotPendingCountsWithoutTouchingAgent`, `TestOpenBotPendingFollowsOutboxLifecycle`, `TestOpenBotPendingQuietHours`. `go vet` e `go test ./...` passam.
- [ ] Deploy no Railway depois do seu OK.

### Fase 2: vigia (opdeals, local)
- [ ] `cmd/atendente-vigia`: loop de 60 s com a decisão acima, configuração por variável de ambiente (`OPDEALS_URL`, `POSTSALE_AGENT_TOKEN`, IDs do agente e da rotina, intervalos), leitura dos arquivos `automation/*` a cada disparo, log, aviso de falha pelo ntfy.
- [ ] Testes da decisão com relógio falso e um servidor falso no lugar do OpenBot e do opdeals (pendente, aberta, batimento, intervalo mínimo, OpenBot fechado, 429).
- [ ] `bot/atendente/vigia.plist` (launchd, `KeepAlive`) e instalação pelo `make atendente`.

### Fase 3: virar a chave
- [ ] Rotina "Atendimento wpp" pausada no OpenBot, ou em `*/30` se a fase 0 der plano B.
- [ ] Texto da rotina: "Se o evento disser só pós-venda, faça os passos 1 a 3 e encerre; batimento ou sem evento, rodada completa." Mesmo texto em `bot/atendente/rotina.md`.
- [ ] `SETUP.md`: seção do vigia (como ver o log, como parar, o que fazer se o OpenBot reiniciar).

### Fase 4: observar 2 dias
- [ ] Rodadas por dia, quantas vazias, tempo entre a venda e a primeira mensagem, e se o cliente orgânico esperou mais de 20 min.
- [ ] Ajustar o batimento (15, 20 ou 30 min).

## Decisões suas
- Batimento para WhatsApp e Instagram orgânicos: 15, 20 (sugestão) ou 30 min. É o tempo máximo que um cliente novo espera.
- Cron pausado (sugestão) ou em `*/30` como rede de segurança.
- Ligar "Local scripts" no agente Atendente. Isso deixa qualquer script do seu usuário no Mac mandar mensagem e rodar rotina desse agente, com o token do arquivo.

## Fora deste plano
- Ler o IndexedDB do WhatsApp Web para contar não lidas.
- Webhook pela nuvem do OpenBot (serve se um dia o servidor precisar acordar o agente direto).
