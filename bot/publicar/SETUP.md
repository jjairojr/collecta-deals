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
- Quem usa o quê: Social `criar_peca`, `pedir_aprovacao` e `situacao`; Estrategista `situacao`; Publicador `inicio`,
  `aprovados`, `publicado` e `falhou`. A trava de cada pasta garante.
- `criar_peca` gera o vídeo (1080x1920, Reels e TikTok) ou a arte (1080x1350) com os modelos `Peca-Story` e `Arte-Feed`
  de `collecta-app/store/motion` (`scripts/peca.mjs`, Chrome do sistema na porta 3123). Preço e foto vêm do catálogo
  do site (`/feed/meta.csv`); o agente só escreve os textos, com limite de letras e sem travessão. Saída em
  `store/motion/out/agente/`, fotos baixadas em `store/motion/public/agente/` (apagadas depois de 3 dias), registro em
  `pecas.log`. Uma peça por vez, até 12 por dia. Variáveis opcionais: `PUBLICAR_MOTION`, `PUBLICAR_CATALOGO`,
  `PUBLICAR_NODE`.
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

## Testes de anúncio no Meta

O Social propõe com `propor_teste` (2 a 5 criativos, objetivo `seguidores` ou `vendas`, público `goiania` ou `brasil`,
verba total de R$ 30 a R$ 150 e 3 a 7 dias). O Jairo recebe o ntfy com a capa do primeiro criativo e os botões
**Aprovar** e **Rejeitar** (`POST /api/aprovar-anuncio/{id}`). O teto é de R$ 150 em 7 dias somando todos os testes que
não foram rejeitados, conferido no dashboard (`internal/adtestq`, `data/adtests.json`).

O `com.collecta.anuncios` (launchd, a cada 30 min) roda `publicar-mcp anuncios`:

- Teste aprovado: cria a campanha, o conjunto com `lifetime_budget` igual à verba aprovada e `end_time`, sobe os
  vídeos e as imagens, cria um anúncio por criativo, tudo pausado, e só no fim liga anúncios, conjunto e campanha.
  Se algo falhar no meio, pausa a campanha, marca o teste como falho e avisa. O arquivo é conferido pelo SHA-256.
- Teste no ar: lê os números por anúncio. Depois de R$ 15 gastos, pausa o criativo sem resultado ou com custo por
  resultado acima do dobro do melhor, sempre deixando 2 ligados. Nunca aumenta verba nem liga nada sozinho.
- Uma hora depois do fim: grava o resumo com o vencedor (o Social lê em `testes_anuncio`) e avisa.
- Seguidores: `OUTCOME_TRAFFIC` com destino perfil do Instagram (`PROFILE_VISIT`), 30 km da loja, 18 a 45 anos,
  medido em cliques para o perfil. Vendas: `OUTCOME_SALES` com o Pixel em `PURCHASE`, Brasil com interesse em Pokémon,
  One Piece e card game, medido em compras, ou visitas à página enquanto não houver compra.
- Registro em `/Users/Shared/collecta-bot/publicar/anuncios.log`; o vigia `com.collecta.saude` avisa se o job parar.

## Reels de teste no Instagram

O Instagram pelo navegador não tem a opção de Reels de teste (conferido em 07/11/2026), então eles saem pela API:

**Como sai (desde 08/11/2026):** o `trial_params` só funciona no envio por link (`video_url`). No envio resumível, os 3 primeiros saíram como reels normais, no perfil. Por isso o job abre um túnel temporário do `cloudflared` (`/opt/homebrew/bin/cloudflared`, ou `META_CLOUDFLARED`) que serve só aquele vídeo, num caminho aleatório, entrega o link ao Instagram e fecha o túnel depois de publicar. Se mesmo assim o reel cair na grade (`is_shared_to_feed` verdadeiro), avisa "Reels de teste caiu na grade" para tirar pelo app (Gerenciar post, Remover da grade do perfil). Liga e desliga com `META_REELS_TESTE=on` no `anuncios.env`.

- O Social pede com `pedir_aprovacao` no canal `instagram`, formato `teste`. O pedido já entra aprovado, como o
  TikTok, e o Jairo recebe só o aviso "Reels de teste agendado pela Collecta" com a prévia.
- O Publicador não vê esses itens em `aprovados`. Quem publica é o mesmo `com.collecta.anuncios` (a cada 30 min):
  abre o túnel, cria o container `REELS` com `video_url` e `trial_params` `{"graduation_strategy":"SS_PERFORMANCE"}`,
  espera o `status_code` `FINISHED` e chama `media_publish`. Registra o link como publicado e avisa "Reels de teste no ar".
- O reel aparece só para quem não segue a conta; se for bem, o Instagram coloca no perfil sozinho.
- Falha passageira volta na rodada seguinte, até 3 tentativas; na última avisa "Reels de teste não saiu".
- Limite da Meta: 100 posts por API em 24 horas.
- Precisa do mesmo `anuncios.env` dos testes de anúncio, com `instagram_content_publish` no token, e do `cloudflared`
  instalado (`brew install cloudflared`).

### Uma vez só

1. **Saldo:** recarregue o pré-pago da conta de anúncios. Sem saldo nada entrega.
2. **App:** em developers.facebook.com, crie um app do tipo Empresa ligado ao portfólio Collecta TCG (ou use um que
   já exista) e adicione o produto **Marketing API**. Para a conta da própria loja não precisa de revisão do app.
3. **Usuário do sistema:** Configurações do negócio > Usuários > Usuários do sistema > Adicionar (função Admin).
   Em "Atribuir ativos", dê controle total da **conta de anúncios**, da **Página** e da **conta do Instagram**
   @tcgcollecta. Em Apps, adicione o app do passo 2.
4. **Token:** no usuário do sistema, "Gerar novo token", escolha o app, validade **Nunca** e marque `ads_management`,
   `ads_read`, `business_management`, `pages_show_list`, `pages_read_engagement`, `instagram_basic` e
   `instagram_content_publish` (esta última é a dos Reels de teste). Copie o token
   direto para o arquivo do passo 6; não mande por chat.
5. **Ids:** conta de anúncios (`act_...`, no Gerenciador de Anúncios), Página (Configurações da Página > id) e o
   id do Instagram (Configurações do negócio > Contas do Instagram > @tcgcollecta). O Pixel já é `959341777245828`.
6. **Ligar:**
   ```sh
   cd ~/Documents/www/opdeals && make anuncios
   ```
   Na primeira vez ele cria `/Users/Shared/collecta-bot/publicar/anuncios.env` (chmod 600) e para. Preencha o
   arquivo (token, ids, `OPDEALS_ADMIN_TOKEN` e o `NTFY_TOPIC` do radar) e rode `make anuncios` de novo.
7. **Primeiro teste assistido:** peça ao Social um teste de seguidores com R$ 30 em 3 dias, aprove e confira no
   Gerenciador de Anúncios, depois de até 30 min, a campanha "Collecta teste ...". O objetivo de visitas ao perfil
   está em liberação gradual no Meta: se a conta não tiver, o teste falha com o motivo no ntfy e nada é gasto.
