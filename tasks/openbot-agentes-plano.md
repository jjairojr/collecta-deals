# OpenBot: regras novas e correções dos agentes (03/11/2026)

Base: auditoria das conversas de 03/11 dos cinco agentes (Pesquisador, Social, Marketplaces, Estrategista, Atendente), dos relatórios gerados, da trava e do banco do OpenBot. Nenhuma rotina de marketing rodou agendada ainda: a primeira é 04/11 às 7h.

## O que foi encontrado

### Comum ao time de marketing
- Trava bloqueia qualquer mensagem que cite "Atendente Collecta"; o papel do Estrategista manda escrever um bloco "para o atendente" no radar. Às 17:08 o Estrategista foi barrado por isso.
- Nenhum relatório tem a seção "Pendências do Jairo" que `instrucoes.md` exige; cada agente inventou um nome. O radar só junta o que achou.
- Os agentes tentam `evaluate`/`run_script` (bloqueados) para esperar ou raspar página: 10 tentativas no dia. Ninguém usa `wait_for`, que é liberado.
- Screenshot 2400x1600 em vez de `snapshot` texto: rodada do Social com 40 MB de transcrição, do Pesquisador com 15 MB.
- Chat longo com regras velhas em contexto: o Pesquisador recusou o admin da Liga 6 vezes (inclusive 3 pedidos do Estrategista) enquanto outra sessão já lia; o radar registrou "o Pesquisador mandou e depois negou". Mensagem de colega cai na sessão principal, que pode estar desatualizada.
- Google Ads: você decidiu deixar de fora (17:02), mas continua em `instrucoes.md`, `papel-social.md` e na rotina.
- `send_message` com `to` + `message` funciona (entregas confirmadas); os agentes erram o nome dos campos e o parâmetro `name` do `tool_call` (4 vezes).
- Sessões criadas antes da trava (13:23) carregaram o `~/.claude/CLAUDE.md` global, `lessons.md` e o plugin caveman. Rodadas agendadas abrem sessão nova e não têm esse problema.

### Pesquisador (haiku)
- Única rodada real (18:07): só a Liga. TikTok, Instagram, Reddit, YouTube, concorrentes: zero. `concorrentes.md` segue vazio.
- "Em alta" com dado de agosto/setembro apesar da regra dos 7 dias; tabela de preços toda de set/2026.
- Cloudflare: a regra manda "parar e pedir ao Jairo"; às 7h a rodada morre. Às 19:32 o desafio passou sozinho em 2 s.
- [CARRINHOS] enviado duas vezes para os mesmos 2 clientes em 12 min: não há registro de "já enviado".
- 5 min chutando filtros de singles (24 `select_option`, 8 erros) até você mandar procurar o ranking. O procedimento certo está em `historico.md` (singles: Cards e Produtos > jogo > Mais Vendidas > Minha Loja, tcg=2/11/19; selados tcg=90 com `search_metrics=2`).
- Pediu `remember` (bloqueado) quando você mandou "coloca na memória".
- Haiku dá conta da Liga com checklist exato; a parte aberta ("descobrir o que está em alta" em 10 fontes) é onde ele deriva.

### Social (sonnet)
- Papel pede o que os painéis não dão: "números de ontem" (Insights só a partir de 7 dias), ROAS (sem evento de compra), Google Ads (sem campanha). O agente gasta tempo e escreve "não há".
- Aba Conteúdo do Instagram (salvos, compartilhamentos, pior post) nunca aberta; comentários/DMs não lidos por decisão própria.
- Queda de 96,7% no TikTok entregue sem investigar (regra 5 exige contar posts); a rodada histórica fez isso 45 min depois: "falta de vídeo, não algoritmo".
- App Store Connect e Play Console deslogados: o resumo de segunda vai falhar.
- O relatório histórico (`historico.md`, "Redes e anúncios") é forte; o diário é 70% descrição.

### Marketplaces (sonnet)
- Nenhuma escrita nos marketplaces (24 cliques, só navegação). `mudancas.md` vazio. Correto.
- O papel cala sobre envios, etiqueta, NF-e, perguntas de comprador, reclamações, devoluções, promoções, Product Ads, "Automatizar preço", Full: o agente acertou por conta própria em deixar os 3 envios para você.
- 8 nomes de avaliadores do Google no arquivo (fere a regra de dados de cliente).
- Sobrescreveu o relatório das 13h; não comparou preço com Liga/TikTok Shop; TikTok Shop deslogado.
- Relatório das 18h útil: 3 envios com prazo, anúncio sem dados fiscais travando despacho, Gem Pack Vol.4 duplicado e "Perdendo", M6 Box com nota 40, 8 avaliações 5,0 sem resposta.

### Estrategista (sonnet)
- Radar com 9.801 caracteres (limite 2.500), ordem antiga, sem "3 números", colagem dos 3 relatórios.
- Radar gravado 2 vezes sem insumo (13:26, 18:09), com ntfy. 5 pushes no dia.
- "ATENÇÃO: números da Liga não verificados" falso: o dado estava certo no arquivo do Pesquisador.
- Repete "Copag não confirmou distribuição nacional" (radar e `campanhas.md`) contra a regra "fato da Collecta vale mais que notícia".
- A estratégia de verdade (4 Reels/semana, CTA único, 1 vídeo TikTok/semana, verba só no formato de R$ 0,31/visita, Meu Negócio, pré-venda OP-18, carrinhos) existe só no chat das 19:11.
- `resultados.md` e `historico.md` "Semanas" vazios; `~/.collecta-atendente/origem.csv` não existe (o semanal lê).
- Horário: Daily 08:00 roda antes do Marketplaces (07:40) e do Pesquisador (40 min) terminarem. SETUP diz 8h30.

### Atendente (sonnet, rodada a cada 5 min)
- Até 20:25 só via as regras até o passo 6 (corte de 4 KB). Estrago de hoje: Susan e Marcos marcados como "falha permanente" porque a busca do WhatsApp não achou o número (regra manda `/send`); `autorizar_envio` chamado nos textos de pós-venda (duplica o histórico); Instagram pulado 2 vezes lendo a regra ao contrário; "Imagina, Nicolas!" respondido a um "obrigado" (regra "fim de papo"); Luciano e Ewerton abordados fora do padrão por ordem avulsa no chat (balões iguais, sem produto).
- Balão 1 do pós-venda não tem variante de pré-venda: "já tá separado" para o Gem Pack Vol. 7, que chega no fim de novembro.
- Três textos diferentes de rotina (OpenBot, `rotina.md`, `instructions.md`); 5 rodadas vazias escreveram parágrafo em vez de `[[no-update]]`.
- Custo: US$ 5,69 hoje (28 rodadas, 71% vazias a US$ 0,10). Com `*/5 8-21` são 156 rodadas: US$ 16 a 20 por dia só olhando a lista.
- Rodada `[[no-update]]` não deixa rastro nenhum (nem .jsonl): não dá para auditar.
- `origem.csv` nunca foi gravado: conferir `registrar_origem`.

## Mudanças

### 1. OpenBot (pela tela)
- [ ] Rotina Daily do Estrategista: 08:00 → 08:45. Weekly: segunda 09:00 → 09:30.
- [ ] Rotina Morning Social: texto cortado ("ça a leitura") → texto inteiro, sem Google Ads.
- [ ] Rotina Atendimento wpp: "WhatsApp e DM do Instagram... Se não houver nada para enviar nem responder, chame routine_no_update e responda apenas [[no-update]]". Cadência: decisão sua (manter 5 min, ou 10 min, ou webhook do pós-venda + varredura a cada 15 min).

### 2. `collecta-app/store/agente-marketing/instrucoes.md` (regras comuns)
- [ ] Nunca escrever o nome "Atendente Collecta" em mensagem, canal ou radar: escreva "o atendente".
- [ ] Seção "Navegador e ferramentas": logado = a URL do painel abre com dados (lista de abas vazia não é deslogado); página sem dados ou "Just a moment" = `wait_for` dom-quiet 8 a 10 s e mais 2 tentativas, depois anote e siga; `snapshot` com `image: never`, screenshot só se o número não vier no texto; `evaluate`, `run_script`, `fill_fields`, `upload_files`, Exportar e Download são bloqueados, não tente; espera entre páginas = `wait_for`; `Glob`/`Grep` sempre com `path` em `store/marketing`; `send_message` é `{to: "Nome do agente", message: "..."}`; em `tool_call` o parâmetro é `name`.
- [ ] Entrega comum: seções fixas por papel, "## Pendências do Jairo" obrigatória em todo arquivo; se o arquivo do dia já existir, releia e reescreva inteiro (sem blocos "atualização"); pt-BR; nada com mais de 7 dias em "Em alta"; tempo máximo por rodada (Pesquisador 40 min, Social e Marketplaces 15, Estrategista 10): ao bater, grave o que tem e mande a mensagem.
- [ ] Login: "usar a conta que o Jairo já deixou logada é permitido; proibido é digitar senha ou código".
- [ ] Pedido do Estrategista dentro do seu papel é para executar; recuse só o que está fora do papel ou quebra regra, dizendo qual. Ao receber mensagem nova, releia as regras antes de agir.
- [ ] Memória: "você não tem memória entre rodadas; procedimento novo vai na seção 'Para o papel' do seu arquivo do dia".
- [ ] Avaliação do Google: identificar por data, nota e primeiras palavras, nunca pelo nome. "Nunca publique..." passa a incluir perguntas, chat e mensagens do ML e do TikTok Shop.
- [ ] Google Ads sai dos canais do Social.

### 3. `papel-pesquisador.md`
- [ ] Cloudflare: `wait_for` e 2 tentativas; nunca esperar o Jairo dentro da rodada; tentar a Liga de novo no fim.
- [ ] Admin: URLs exatas de Minha Loja (`search_metrics=2`), selados tcg=90, singles por jogo tcg=2/11/19 (Cards e Produtos > jogo > Mais Vendidas > Minha Loja), período mensal e anual. Todo dia: top 5 selados e top 5 singles por jogo (Minha Loja, mês atual) e top 5 da plataforma por jogo.
- [ ] Carrinhos: dedup em `relatorios/carrinhos-enviados.md` (data | id do cliente na URL do cadastro | valor; sem nome nem telefone). Já listado não vai de novo. Se a lista chegar fora da rotina, o atendente trata na próxima rodada.
- [ ] Fontes em ordem com tempo: Liga 15 min; lançamentos 5; TCGplayer 5; TikTok/Instagram/concorrentes 10 (se pedir login, "pede login" e segue).
- [ ] `wait_for` 7 s entre páginas do admin.

### 4. `papel-social.md`
- [ ] Item 1: últimos 7 dias contra os 7 anteriores, com as URLs do Insights, da aba Conteúdo (salvos, compartilhamentos, melhor e pior post) e do TikTok Studio.
- [ ] Item 2: Meta Ads 7 dias por campanha (gasto, CPM, CPC, CTR, custo por resultado); ROAS só se houver coluna de compras, senão "sem evento de compra" em Pendências. Google Ads sai.
- [ ] Item 3: ler só a prévia da caixa de entrada; nunca abrir conversa.
- [ ] Item 5: ler o relatório anterior e comparar; queda > 30% = contar posts da semana e da anterior, causa e ação em 2 linhas.
- [ ] Modelo fixo do arquivo e mensagem ao Estrategista em 3 linhas. Segunda: App Store Connect e Play Console só se logados.

### 5. `papel-marketplaces.md`
- [ ] Bloco "Só leitura, nunca mexa": envios, etiquetas, NF-e, perguntas, mensagens, reclamações, devoluções, cancelamentos, promoções, cupons, Product Ads, Automatizar preço, Full, reputação; TikTok Shop pedidos e chat; Merchant Center feed e configurações. Tudo isso vira linha em Pendências do Jairo.
- [ ] Todo dia: para cada anúncio "Perdendo" no ML, comparar com o piso da Liga e do TikTok Shop; item "Pendências do Jairo".
- [ ] "Como abrir cada painel": URLs do ML (vendas e anúncios), Seller Center, Merchant Center e Meu Negócio; como reconhecer deslogado.
- [ ] "fotos que já estão no anúncio" vira "ordem das fotos já no anúncio (nunca envie arquivo)".

### 6. `papel-estrategista.md`
- [ ] Modelo fixo do radar com teto por seção: Ação de hoje (canal, peça, horário, meta numérica, como medir em 48 h) ≤ 600; 3 números contra referência ≤ 450; Vendas x divulgação ≤ 300; Em alta (7 dias) ≤ 300; Acompanhamento das ações anteriores ≤ 300; Pendências do Jairo ≤ 400. Contar os caracteres antes de gravar; passou de 2.500, cortar nesta ordem: Em alta, Vendas x divulgação, Acompanhamento.
- [ ] Insumo vale se foi gravado hoje depois da rotina do agente. Faltou ou está velho: uma mensagem, espera até 10 min, depois "Sem dado hoje: <agente>" e segue. Nunca copiar números de rodada anterior como de hoje. Resposta "blocked" vira Pendência; não reenvia.
- [ ] ntfy: um por rodada com o radar inteiro; no máximo 2 por dia.
- [ ] Novo `store/marketing/plano.md` (só o Estrategista edita, ≤ 1.500 caracteres): estratégia vigente, metas da semana, o que não fazer, cada item com o número que justifica. Semente: a resposta das 19:11. O radar aponta para ele; segunda revisa.
- [ ] Referência: `historico.md` "Semanas" quando existir; até lá, o arquivo do dia anterior ou `historico-margem.md`, dizendo qual usou. Não empurrar produto com margem abaixo de 15%.
- [ ] Segunda: `origem.csv` ausente = "sem arquivo" e segue; `resultados.md` ganha uma linha por ação dos radares da semana, com o número medido ou "não medido".
- [ ] Tirar a frase "Copag não confirmou" de `campanhas.md` (contradiz a pré-venda PT-BR em andamento) ou você confirma o fato.

### 7. `guard.sh` e `SETUP.md` (collecta-app)
- [ ] `check_write`: liberar `store/marketing/plano.md` para o estrategista.
- [ ] Mensagem da linha 102: "tool_call <alvo> não permitido para <papel>" (hoje diz "não usa navegador" para qualquer coisa).
- [ ] A checagem do atendente continua no payload inteiro (falha fechada); o conserto é a regra de não escrever o nome.
- [ ] SETUP: tabela de rotinas com 08:45 e 09:30; "não cole a rotina no chat longo, use Rodar agora"; logar App Store Connect e Play Console ou tirar do papel.

### 8. Atendente (`cmd/atendente-mcp/instructions.md`, `bot/atendente/rotina.md`, código)
- [ ] Pós-venda: não chamar `autorizar_envio` nos textos já aprovados; `permanente` só quando o WhatsApp disser que o número não tem WhatsApp (busca não acha contato novo: sempre `/send`); balão 1 com variante de pré-venda ("já tá reservado, te aviso quando chegar").
- [ ] Instagram 1 reescrito sem ambiguidade: "se não houver aba, abra o inbox com navigate".
- [ ] Carrinhos: lista fora da rotina é tratada na próxima rodada, não por ordem avulsa.
- [ ] Navegador: "para ler a tela use snapshot; run_script é bloqueado".
- [ ] `rotina.md` igual ao texto do OpenBot.
- [ ] Código: `posvenda_confirmar` não duplica o histórico quando `autorizar_envio` já gravou; conferir por que `registrar_origem` não gera `origem.csv`.
- [ ] Depois (tarefa própria): rastro de cada rodada (`rodadas.log`) e gatilho por webhook para o pós-venda em vez de cron.

## Pendências suas
- Susan e Marcos: o pós-venda deles foi marcado como falha permanente por engano. Posso destravar na prod para a próxima rodada tentar de novo (precisa do seu OK, é escrita na prod).
- 17 pedidos da Liga aguardando envio (R$ 6.465) e 3 vendas do ML para enviar, uma travada por anúncio sem dados fiscais.
- Logar TikTok Shop (Seller Center), App Store Connect e Play Console nos navegadores do Marketplaces e do Social.
- Cadência do atendente (5 min, 10 min, ou webhook + 15 min).
- Pixel/evento de compra no Meta, se quiser ROAS.

## Revisão

03/11/2026, 21h: aplicado.

- `collecta-app/store/agente-marketing/`: `instrucoes.md` reescrito (seções "Navegador e ferramentas" e "Entrega: o arquivo do dia", regra do nome do atendente, pedido do Estrategista dentro do papel, releitura das regras a cada mensagem, login com conta já logada, Google Ads fora, avaliações sem nome). Os quatro `papel-*.md` reescritos com seções fixas, "Pendências do Jairo" obrigatória, tempos, URLs exatas da Liga (Minha Loja, singles por jogo), dedup de carrinhos em `relatorios/carrinhos-enviados.md`, radar com teto por seção e `plano.md`. `guard.sh`: `plano.md` liberado só para o Estrategista, mensagens de bloqueio claras; testado com 9 entradas falsas (`sh -n` ok). `SETUP.md`: horários, aviso de não colar rotina no chat, logins. `store/marketing/plano.md` criado (v1 com a estratégia das 19:11).
- `opdeals`: `cmd/atendente-mcp/instructions.md` (passo 6 com `[[no-update]]`, navegador com `snapshot` e `/send`, balão 1 de pré-venda, `permanente` só quando o WhatsApp disser, sem `autorizar_envio` no pós-venda, Instagram 1 sem ambiguidade, carrinhos mesmo com pedido avulso). `history.go` ignora texto repetido para o mesmo número em 10 min (teste `TestHistoryIgnoresRepeatWithinTenMinutes`), `go test` ok, `make atendente` instalado às 21:00. `bot/atendente/rotina.md` igual ao OpenBot.
- OpenBot (pela tela, conferido no banco): "Atendimento wpp" com WhatsApp + Instagram e `[[no-update]]`; "Morning Social" com o texto inteiro, sem Google Ads; "Daily" do Estrategista em `45 8 * * *` (próxima 04/11 8h45). "Weekly" em `30 9 * * 1` (próxima 09/11 9h30). Nenhum agente ficou com o campo Title preenchido (um texto caiu lá por engano e foi limpo).
- `origem.csv` não existe porque `registrar_origem` nunca foi chamado (a regra "como nos achou" ficava depois do corte das 4 KB); não é bug de código.
- A rodada das 21:00 do atendente ainda rodou com o binário anterior (regras `defdf7f2`); respondeu 1 DM do Instagram, prova de que a seção Instagram agora chega inteira.
- 21h50: pós-venda de Susan (12380826), Marcos (12380950) e Carlos (12380196) reaberto na prod (conversas de `closed/failed` para `queued`, oferta de volta à fila) subindo `conversations.json` corrigido para o volume do Railway; o Jairo rodou o script porque o modo automático barrou a escrita. O atendente envia na primeira rodada fora do horário silencioso.
- Bug achado no servidor: `handleAgentFailed` fecha a conversa em qualquer falha da oferta, mesmo com `permanent = false` (o campo só vale para o rastreio). A regra nova do atendente ("permanente = false para a próxima rodada tentar de novo") depende de corrigir isso em `internal/api/postsale_agent.go` e subir no Railway.
- Não feito: cadência do atendente (continua 5 min), rastro de rodada `[[no-update]]`, webhook do pós-venda, correção acima no servidor.
