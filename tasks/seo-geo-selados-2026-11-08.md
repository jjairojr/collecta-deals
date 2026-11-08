# SEO / GEO / AEO da vitrine — foco em selados e termos amplos (2026-11-08)

Foco do dono: selados e termos como "loja TCG", "loja de card games", "loja de pokémon", "cartinhas pokémon". Nome de single fora de escopo.

## Fase 1 — correções
- [x] Selado esgotado com gêmeo vivo (mesmo jogo + nome normalizado + idioma compatível) redireciona 301 para o vivo; esgotados gêmeos entre si viram um só
- [x] Decks ST-31/33/34/35/36 antigos (nome diferente) entram em legacy-slugs.json; legacy-slugs passa a valer também quando o slug antigo ainda existe como esgotado
- [x] Headline sem "()" vazio ("Elite Trainer Box Coleção Treinador Avançado () - …")
- [x] Imagens de selados e acessórios pelo próprio domínio (/foto, allowlist de hosts, webp), tirando o host do Railway e o CDN da Liga do HTML
- [x] Organization.sameAs com Facebook, loja na Liga e MyPCards
- [x] Barra de anúncio: "CARTA CERTIFICADA" vira "CARTA ORIGINAL"
- [x] /llms.txt com quem somos, loja física, páginas principais, coleções e selados em estoque com preço
- [x] IndexNow: chave pública + cron diário da Vercel avisando Bing das páginas de selado/categoria (precisa CRON_SECRET na Vercel)

## Fase 2 — páginas que respondem
- [x] lib/sealed-guide.ts: coleções seladas (Reinado Delta, Escuridão Absoluta, Caos Ascendente, GEM Pack, Pokémon Center JP, Starter Decks ST-31 a ST-36, Vendetta) com fatos checados e "o que vem" por formato
- [x] /selado/[jogo]/[colecao]: página da coleção com produtos, preço, o que vem em cada um, perguntas frequentes, breadcrumb, ItemList, FAQPage (só com 2+ produtos vivos; senão redireciona para o jogo)
- [x] Página do selado: bloco "sobre a coleção" + "o que vem", outros produtos da coleção, perguntas frequentes, schema com additionalProperty/releaseDate/descrição rica
- [x] /selado e /selado/[jogo]: H1 melhor, intro, links de coleções, guia "qual selado comprar", FAQ, breadcrumb, ItemList
- [x] /cartinhas-pokemon: guia de onde comprar cartinhas Pokémon originais, com selados em estoque
- [x] Home: título/descrição/H1 com "card games" e "cartinhas", FAQ da loja com FAQPage; /loja com "loja de card games"
- [x] Sitemap com coleções e /cartinhas-pokemon; footer com links

## Verificação
- [x] tsc sem erro
- [x] next build contra a API de prod
- [x] next start local: conferir redirects das duplicatas, páginas novas, JSON-LD, /llms.txt, /foto

## Pendências do dono
- CRON_SECRET na Vercel para o IndexNow
- Deploy (nada commitado)

## Revisão (2026-11-08)
- Build de produção passou (lint + tipos) contra a API de prod; servidor local conferido.
- Duplicatas: Fukuoka (2 cópias), GEM Pack Vol.4 CN, ST-31, Parceiro Inicial e Vendetta antigas respondem 308 para a página viva; ST-32 (sem gêmeo vivo) continua como esgotada.
- 8 páginas de coleção geradas; Vendetta (1 produto) redireciona 307 para /selado/riftbound.
- HTML da página de selado sem nenhuma ocorrência do host do Railway; /foto devolve webp e recusa host fora da lista e id inválido (404).
- /api/indexnow responde 503 sem CRON_SECRET; chave pública servida em /8dcbe9217ff8190a0eeac3fa63ed1bac.txt.
