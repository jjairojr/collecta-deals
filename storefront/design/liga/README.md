# Assets da loja Collecta na LigaMagic

A Liga valida cada upload contra uma largura/altura exata, então cada peça é
renderizada no tamanho final — nada aqui escala. O render usa Chrome headless em
`--force-device-scale-factor=1`, com as fontes reais da marca (Baloo 2 ExtraBold,
Press Start 2P, DM Sans) e o mascote, todos já versionados em
`storefront/app/og/`.

```
npm/node build.mjs   # gera as peças em dist/
node proof.mjs       # gera dist/PROVA-collecta-liga.png (folha de aprovação)
```

Loja: `id=866280` · <https://www.ligamagic.com.br/?view=ecom/home&id=866280>

## Onde cada arquivo entra

| arquivo | tela do admin | spec da Liga |
| --- | --- | --- |
| `logo-200x85.png` | Layout → Configurar Layout Responsivo → Logo | 200×85 · 80kb · jpg/png |
| `header-bg-1920x110.png` | idem → Background do Cabeçalho | 1920×110 · 300kb |
| `noticia-hero-1170x360.png` | Layout → Página Inicial → Imagem da Notícia | 1170×360 · 200kb |
| `banner-superior-1170x60.png` + `banner-superior-mobile-400x80.png` | Layout → Banners → Banner Superior | 1170×60 e 400×80 |
| `banner-full-confianca-1170x275.png` | Layout → Banners → Banner 100% | 1170×275 |
| `banner-triplo-{singles,selados,acessorios}-400x275.png` | Layout → Banners → Banner Triplo (posições 1/2/3) | 400×275 |
| `miniatura-300x300.jpg` | Layout → Configurações de sua Loja → Miniatura para a URL | 300×300 · 80kb · **jpg** |
| `marketplace-logo-101x30.jpg` | idem → Logo no Marketplace | 101×30 · 40kb · **jpg** |
| `marketplace-avatar-55x55.jpg` | idem → Avatar Marketplace | 55×55 · 10kb · **jpg** |
| `logo-200x85.png` | idem → Painel Administrativo → Logo | 100–210 × 85 · 85kb |
| `favicon-{16,32,48,192,512}.png` | idem → Ícones de Favorito | ico/png |

`banner-hero-1170x275.png` é um hero alternativo para o slot Banner 100%, gerado
mas não publicado — o slot está com a faixa de confiança.

## Decisões que valem lembrar

- **O background do cabeçalho é renderizado com `background-repeat: repeat` e
  `background-size: auto`**, então ele lada em telas > 1920px. Todo período
  horizontal da arte divide 1920 (`TILE = 96`) e nada depende da posição x — sem
  gradiente horizontal, sem diagonal, sem glow radial. Só assim a emenda é
  invisível.
- **O logo é PNG com transparência e a Liga renomeia o arquivo para `.jpg`, mas
  preserva o alfa.** Não trocar por JPG achatado: o logo precisa deixar o grid do
  cabeçalho passar por trás.
- **Favicon é um sistema responsivo.** De 16 a 48px o traço de tinta do mascote
  virava uma mancha, então esses tamanhos levam a marca tipográfica (quadrado
  rosa + "C" em Baloo). 192 e 512 levam o mascote.
- **A Liga não tem variante mobile para o slot Banner 100%** — a faixa inteira é
  escalada para ~37% no celular. Por isso os títulos são grandes e cada linha de
  apoio cabe em uma linha só (`white-space: nowrap`).
- **O logo do marketplace (101×30) é um mini-cabeçalho com o mascote.** A
  primeira versão era uma placa rosa chapada e ficou feia nas linhas de seller
  do marketplace; a segunda, wordmark em fundo branco, ficou genérica. A que
  valeu: royal + grid + ticks rosa, disco do mascote e wordmark a 13.5px.
- **A Liga corta ~3px da base do logo do marketplace ao exibir.** Por isso a
  peça não tem barra de rodapé e nada crítico fica a menos de 4px da borda
  inferior (disco de 24px puxado 1px pra cima).
- **O rosto do mascote só lê em disco pequeno com crop na cabeça.** O
  enquadramento da arte inteira vira mancha de cabelo, e fechar demais na cara
  (190%) ficou "cheio" demais — o aprovado é `width:155%;left:-26%;top:-42%` +
  `contrast(1.55) saturate(1.25)`: cabeça com o afro de moldura e olhos
  legíveis.
- **A cor "principal do site" da Liga é o acento** (botões, títulos, badges,
  breadcrumb) → rosa `#F6559B`. A "secundária" é a fonte/ícone do cabeçalho →
  branco.
- **Topo e rodapé compartilham UM campo de cor** no admin responsivo. Está azul
  royal; não existe jeito de ter topo escuro e rodapé rosa como na vitrine.
- **O admin da Liga grava só um arquivo por submit.** Enviar vários de uma vez
  faz o save inteiro abortar (o erro aparece como falha de favicon, mas derruba
  também os textos). Um upload por "Salvar".
