# Publicador Collecta (OpenBot)

O Social Collecta escreve a peça e pede aprovação com `pedir_aprovacao`. O Jairo recebe um ntfy com o vídeo ou a imagem
anexados (até 15 MB), a legenda, o horário e os botões **Aprovar** e **Rejeitar**. O Publicador Collecta roda de hora em
hora, pega o que foi aprovado e já chegou no horário, posta pelo navegador no Instagram ou no TikTok Studio, confere e
manda o link no ntfy.

- Fila no dashboard (`internal/publishq`, `data/publishq.json` no Railway). Rotas `/api/publish` com o admin token e
  `POST /api/aprovar/{id}?acao=aprovar|rejeitar&t=<código>` pública, com um código secreto por peça (é o que os botões
  do ntfy chamam). Prévia de link no celular não aprova: só POST.
- Servidor de ferramentas `collecta-publicar` (`cmd/publicar-mcp`). Regras do Publicador em
  `cmd/publicar-mcp/instructions.md`, entregues pela ferramenta `inicio`.
- O arquivo é conferido pelo SHA-256 na hora do pedido e de novo antes de postar. Se mudou depois da aprovação, a
  peça falha e o Jairo é avisado.
- O Publicador só consegue enviar arquivo de `/Users/Shared/collecta-bot/publicar/aprovados/<id>/`, que só recebe o
  que foi aprovado. Ele só abre `instagram.com` e `tiktok.com` (sem DM, conta, anúncio, live), não roda script, não
  pede aprovação e não fala com outro agente.
- Quem usa o quê: Social `pedir_aprovacao` e `situacao`; Estrategista `situacao`; Publicador `inicio`, `aprovados`,
  `publicado` e `falhou`. A trava de cada pasta garante.
- Registro: `/Users/Shared/collecta-bot/publicar/publicacoes.log` e `bloqueios.log`.
- Fora por enquanto: story, música da biblioteca do Instagram e carrossel com música.

## Uma vez só

1. **Subir o dashboard** com a fila nova (push na main do opdeals; o Railway faz o deploy). Confira:
   `curl -s -o /dev/null -w '%{http_code}' -X POST "$OPDEALS_URL/api/aprovar/x?acao=aprovar&t=x"` tem que dar `404`.
2. **Instalar:**
   ```sh
   cd ~/Documents/www/opdeals && make publicar
   ```
   Rode de novo sempre que mudar algo em `cmd/publicar-mcp` ou `bot/publicar`.
3. **Criar o agente** "Publicador Collecta" no OpenBot:
   - Provedor: Claude Code. Acesso: **Acesso total** (é o único modo em que o OpenBot carrega a trava da pasta).
   - Navegador: perfil **Só este agente**.
   - Instruções: `Siga as instruções do servidor collecta-publicar.`
   - Não use esse agente para mais nada e não o ponha em canais do OpenBot.
   - Trava na pasta do agente:
     ```sh
     A=~/OpenBot/Agents/<id-do-agente-novo>
     mkdir -p "$A/.claude" && cp /Users/Shared/collecta-bot/publicar/agente/.claude/settings.json "$A/.claude/"
     ```
4. **Adicionar o MCP** no painel MCP do OpenBot (é global: a trava de cada agente decide quem usa):
   - Nome: `collecta-publicar`
   - Comando: `/Users/Shared/collecta-bot/publicar/publicar-mcp`
   - Variáveis:
     - `OPDEALS_ADMIN_TOKEN=<ADMIN_TOKEN de scripts/deploy.env>`
     - `NTFY_TOPIC=<o tópico do radar do marketing, o mesmo do Estrategista>`
5. **Logar o navegador do Publicador.** No chat dele: "Abra https://www.instagram.com/" e faça o login do @tcgcollecta;
   depois "Abra https://www.tiktok.com/tiktokstudio/upload" e faça o login do Collecta TCG. Nunca digite senha pelo chat.
6. **Testar a trava.** No chat do Publicador: "rode ls no terminal", "abra https://www.instagram.com/direct/inbox/" e
   "chame pedir_aprovacao". Os três têm que ser bloqueados (`cat /Users/Shared/collecta-bot/publicar/bloqueios.log`).
7. **Primeiro post assistido.** Peça ao Social uma peça com arquivo pronto, aprove no celular e, no horário, rode a
   rotina do Publicador com "Rodar agora", olhando a janela do navegador dele.
8. **Agendar.** Rotina do Publicador de hora em hora, das 9h às 22h, com o texto de `rotina.md`. "Manter computador
   acordado" ligado.

## Onde olhar quando algo der errado

- `publicacoes.log` (o que saiu e o que falhou), `bloqueios.log` (o que a trava barrou), o ntfy.
- A fila inteira: `curl -s -H "X-Admin-Token: $ADMIN_TOKEN" "$OPDEALS_URL/api/publish"`.
- Pedido parado em "esperando o dono": o Jairo não tocou em Aprovar. O botão só funciona uma vez.
- Falha passageira volta na próxima rodada, até 3 tentativas; depois disso, ou com `permanente`, a peça sai da fila e
  o Social precisa pedir de novo.
