Você vai terminar de colocar no ar o agente de baixa de vendas da Collecta no OpenBot e deixar tudo redondo para mim.
Trabalhe no repositório ~/Documents/www/opdeals. Responda em português.

## Contexto

- O OpenBot roda agentes Claude Code neste Mac. Criei o agente "Estoquista Collecta"; a pasta dele é
  ~/OpenBot/Agents/agent-72639cbe-7b7d-469e-bdbb-2eeadd8d468c (modelo Opus 5.5, Acesso total, navegador "Só este agente",
  instrução "Siga as instruções do servidor collecta-baixa.").
- Ele dá baixa no dashboard de produção das vendas da LigaMagic, executa a fila Sincronizar e manda o pós-venda da Liga.
  O MyPCards foi desativado em 03/11/2026: não mexa nele.
- O agente fala com o dashboard só pelo MCP `collecta-baixa` (código em cmd/baixa-mcp, binário instalado por `make baixa`
  em /Users/Shared/collecta-bot/baixa). As regras dele estão em cmd/baixa-mcp/instructions.md. A trava fica em bot/baixa/
  (settings.json + guard.sh, copiados para a pasta do agente). O passo a passo está em bot/baixa/SETUP.md e o plano em
  tasks/baixa-openbot.md.
- A conversa do agente fica em ~/.claude/projects/-Users-jjairo-OpenBot-Agents-agent-72639cbe-7b7d-469e-bdbb-2eeadd8d468c/*.jsonl.
  Os relatórios ficam em ~/.collecta-baixa/<carimbo>.md, o histórico em ~/.collecta-baixa/runs.log e os bloqueios da trava
  em ~/.collecta-baixa/bloqueios.log.
- A 1ª rodada (03/11 19:48) deu certo: baixou o Mr.3 OP16-056 (4 un, R$ 156,24, pedido Liga #12385475). Mas deixou 2
  itens do mesmo pedido como pendência sem motivo real (Brook OP15-032 e Boa Hancock OP16-032 têm dois lotes idênticos).
  Já corrigi as regras (lotes idênticos = baixa do lote com mais unidades) e criei a ferramenta `registrar_pendencia` +
  `pendencias_abertas` no `inicio`.

## O que falta, em ordem

1. Feito em 03/11: Brook OP15-032 e Boa Hancock OP16-032 do pedido #12385475 já foram baixados à mão (1 un cada, lotes
   d9021326 e 73d6cc62 de 4→3, flag da Liga 4→3). Não baixe de novo.
2. Me guiar para reiniciar o MCP `collecta-baixa` no painel MCP do OpenBot (desligar e ligar). O teste de conexão tem que
   listar 11 ferramentas, incluindo `registrar_pendencia`.
3. Acompanhar uma 2ª rodada manual. Eu colo no chat do agente o texto de bot/baixa/rotina.md. Enquanto roda, leia o .jsonl
   mais novo da conversa do agente e o bloqueios.log. No fim, compare o ledger de prod (os 3 jogos: onepiece, pokemon,
   riftbound) com uma foto tirada antes e confira cada baixa: pedido, comprador, qty, total = unitário × qty e flag da Liga.
   Me diga em poucas linhas o que estava certo e o que não estava.
4. Se a rodada estiver certa, me guiar para criar a rotina no OpenBot: 9h, 14h e 20h, texto de bot/baixa/rotina.md,
   "Manter computador acordado" ligado. Depois desligue o cron antigo:
   `launchctl unload ~/Library/LaunchAgents/com.collecta.baixa-vendas.plist`.
5. Se algo der errado no agente, corrija em cmd/baixa-mcp/instructions.md ou no código, rode `go test ./cmd/baixa-mcp/` e
   `make baixa`, copie /Users/Shared/collecta-bot/baixa/agente/.claude/settings.json para a pasta do agente e me peça para
   reiniciar o MCP.
6. No fim, atualize a seção Revisão de tasks/baixa-openbot.md e o bot/baixa/SETUP.md, e me entregue um resumo curto:
   o que está no ar, onde olho quando algo der errado e o que ainda depende de mim.

## Regras

- Nunca faça commit nem push.
- Nunca grave na prod (POST/PUT/DELETE no dashboard) sem a minha confirmação explícita naquele momento. Só leitura é livre.
- Nunca mostre o admin token nem dados pessoais de cliente (CPF, telefone, endereço, e-mail).
- Não navegue na LigaMagic você mesmo: ela tem Cloudflare e já baniu meu IP por excesso de acessos. Quem usa a Liga é o
  agente.
- Não comente código e não deixe variável sem uso.
