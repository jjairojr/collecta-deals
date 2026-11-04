# Baixa Collecta (OpenBot)

Um agente do OpenBot (Claude Code com o seu plano) dá baixa das vendas da Liga no dashboard de produção, executa a fila
Sincronizar e manda o pós-venda da Liga. Ele substitui o launchd `com.collecta.baixa-vendas`, que parou em 28/10/2026 (o
macOS bloqueia o node na pasta Documents) e foi descarregado em 03/11/2026. O MyPCards foi desativado em 03/11/2026: o
agente não abre mais o MyP.

No ar desde 03/11/2026: agente "Estoquista Collecta", pasta
`~/OpenBot/Agents/agent-72639cbe-7b7d-469e-bdbb-2eeadd8d468c`.

- O agente usa o próprio navegador do OpenBot, logado por você na Liga e no MyPCards.
- Ele fala com o dashboard só pelo `baixa-mcp` (este repo, `cmd/baixa-mcp`). O admin token fica no MCP, nunca no
  agente. As regras que ele segue estão em `cmd/baixa-mcp/instructions.md`.
- A trava da pasta (`settings.json` + `guard.sh`) bloqueia terminal, arquivos, web, subagentes e skills. Ela só deixa o
  navegador abrir `ligamagic.com.br`, `ligacollectatcg.com.br`, `ligaonepiece/ligapokemon/ligariftbound.com.br` e
  `mypcards.com`. Upload, download, senha e cofre ficam bloqueados.
- `run_script` fica liberado nesses domínios, porque a leitura dos pedidos e a gravação no MyP e na Liga dependem dele.
- O relatório de cada rodada vai para o ntfy e fica em `~/.collecta-baixa/<carimbo>.md`. O histórico curto fica em
  `~/.collecta-baixa/runs.log`.

## Uma vez só

1. **Instalar:**
   ```sh
   cd ~/Documents/www/opdeals && make baixa
   ```
   Isso põe o binário, a trava e a rotina em `/Users/Shared/collecta-bot/baixa`. Rode de novo sempre que mudar algo em
   `cmd/baixa-mcp` ou `bot/baixa`.
2. **Criar o agente** "Baixa Collecta" no OpenBot:
   - Provedor: Claude Code.
   - Acesso: **Acesso total**. É o único modo em que o OpenBot carrega a trava da pasta.
   - Navegador: perfil **Só este agente**.
   - Instruções: `Siga as instruções do servidor collecta-baixa.`
   - Não use esse agente para mais nada e não o ponha em canais do OpenBot.
3. **Colocar a trava na pasta do agente:**
   ```sh
   ls -t ~/OpenBot/Agents/ | head -3
   A=~/OpenBot/Agents/<id-do-agente-novo>
   mkdir -p "$A/.claude" && cp /Users/Shared/collecta-bot/baixa/agente/.claude/settings.json "$A/.claude/"
   ```
4. **Adicionar o MCP** no painel MCP do OpenBot. O painel é global: todo agente Claude Code do OpenBot recebe todos os
   MCPs ligados (o atendente também abre o `collecta-baixa`, e este agente abre o `collecta-loja`). Quem isola é a trava
   de cada pasta: o `ferramentas-permitidas.txt` do atendente só libera `collecta-loja`, e o desta baixa só libera
   `collecta-baixa`. Nunca crie agente sem trava enquanto esses MCPs estiverem ligados.
   - Nome: `collecta-baixa`
   - Comando: `/Users/Shared/collecta-bot/baixa/baixa-mcp`
   - Variáveis:
     - `OPDEALS_ADMIN_TOKEN=<ADMIN_TOKEN de scripts/deploy.env>`
     - `NTFY_TOPIC=collecta-restock-d2bddd0c5b` (o mesmo tópico da baixa antiga e do restock)
5. **Logar o navegador do agente.** No chat do agente: "Abra https://www.ligapokemon.com.br/?view=ecom/admin/home".
   Faça o login da Liga você mesmo na janela que abrir. A sessão da Liga é por host, e as regras usam
   `www.ligapokemon.com.br`: logar em outro host (ligamagic) deixa o agente deslogado. Depois: "Abra https://mypcards.com/CollectaTCG/vendas" e faça o
   login do MyPCards. Nunca digite senha pelo chat. Se aparecer o desafio do Cloudflare, espere ele passar sozinho.
6. **Testar a trava.** No chat do agente, peça "rode ls no terminal", "leia o arquivo ~/.ssh/config" e "abra
   https://google.com". Os três têm que ser bloqueados (`cat ~/.collecta-baixa/bloqueios.log`).
7. **Primeira rodada assistida.** Cole o texto de `rotina.md` no chat e acompanhe a rodada. Confira as baixas no
   dashboard (Vendas) e o relatório no celular. Se o agente for bloqueado em alguma ferramenta de navegador legítima, o
   nome aparece em `bloqueios.log`. Ajuste o `guard.sh` e rode `make baixa` de novo.
8. **Agendar.** Rotina nova no OpenBot às 9h, 14h e 20h, com o texto de `rotina.md`. Deixe "Manter computador
   acordado" ligado e o Mac na tomada. Com a tampa fechada ou o Mac dormindo, a rodada não acontece.
9. **Desligar o cron antigo**, para as duas não rodarem juntas se o macOS voltar a deixar:
   ```sh
   launchctl unload ~/Library/LaunchAgents/com.collecta.baixa-vendas.plist
   ```

## Editar o que a baixa sabe

- **Regras e mecânica:** `cmd/baixa-mcp/instructions.md`. O agente recebe esse texto no campo `regras` do `inicio`,
  não nas instruções do servidor: o Claude Code corta instruções de MCP em ~4 KB. Depois de mudar, rode `make baixa`. Não precisa reiniciar
  nada no OpenBot: cada conversa abre um `baixa-mcp` novo com o binário instalado. A `versao_regras` no cabeçalho do
  relatório confirma qual versão rodou.

## Onde olhar quando algo der errado

- Relatório da rodada: `~/.collecta-baixa/<carimbo>.md` (e o ntfy). Histórico curto: `~/.collecta-baixa/runs.log`.
- Ferramenta barrada pela trava: `~/.collecta-baixa/bloqueios.log`.
- Conversa inteira do agente: o `.jsonl` mais novo em
  `~/.claude/projects/-Users-jjairo-OpenBot-Agents-agent-72639cbe-7b7d-469e-bdbb-2eeadd8d468c/`.
- Rotinas e MCPs que o OpenBot enxerga (só leitura):
  `sqlite3 "file:$HOME/Library/Application Support/OpenBot/openbot.db?mode=ro" "select name, active from projection_agent_routines"`.
  Não grave nesse banco: as tabelas `projection_*` são reconstruídas a partir de eventos.
- As skills `baixa-vendas`, `liga-vendas` e `myp-vendas` continuam valendo para rodada à mão no Claude Code. Aprendizado
  novo vale nos dois lugares.

## Saber

- O agente não cria lote extra e não reduz o dashboard. Liga acima ou abaixo do dashboard vira pendência no relatório.
- Pedido ambíguo (dois registros possíveis, item que não casa pelo nome) nunca é gravado: vira pendência.
- `URGENTE` no resumo é venda dupla (o mesmo item vendido na Liga e no MyP desde a última rodada). Chega com prioridade
  alta no ntfy.
- Rodada bloqueada pelo Cloudflare para aquele canal e segue no outro. A fila Sincronizar espera a próxima rodada.
- Os riscos: o navegador do agente fica logado na sua conta da Liga e do MyP e pode rodar script nessas páginas. Ele não
  tem nenhum segredo (o token fica no MCP), mas um script nessas páginas enxerga o que a sua sessão enxerga.
