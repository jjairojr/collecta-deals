# Brief: redesign "arcade" do dashboard (web/)

> Cole este arquivo (ou peça pro agente lê-lo) no início de um chat novo.
> Objetivo: redesenhar o dashboard interno seguindo a MESMA linguagem visual "arcade"
> já aplicada na vitrine pública (`storefront/`). É um **RESKIN** — manter toda a
> funcionalidade, dados e lógica; trocar só a camada visual.

## Leia primeiro (referência canônica do estilo)

- `storefront/app/globals.css` — tokens + helpers do design system arcade (fonte da verdade)
- `storefront/components/` — como o estilo é aplicado (`SingleCard`, `chrome/Header`/`TabBar`,
  `ui/Stepper`, `ui/ArtPlaceholder`, `product/SingleDetailView` com as barras de histórico)
- `storefront/design/design_handoff_collecta_marketplace/README.md` — o handoff original
  (tokens, regras, "arcade sticker", tipografia, efeitos)
- `~/.claude/CLAUDE.md` e `~/.claude/lessons.md` — regras de trabalho (ler no começo da sessão)

## O que é o alvo (o dashboard)

- App **React + Vite + TypeScript + Tailwind (v4 `@theme`)** em `web/`.
- Shell atual: sidebar (`AppSidebar.tsx`) + `TopBar.tsx` + conteúdo (`SidebarInset`) +
  `PageHeader.tsx` por view. Config de nav e cores por jogo em `brand.tsx`. Tema em `index.css`.
- Views/abas: **Deals** (`DealsTable` + filtros + grid), **Tracking** (`TrackingPage` — KPIs,
  vendas, price movers, leaderboard de lojas), **Sealed**, **Portfolio** (`PortfolioPage` — KPIs,
  holdings, sold), **Orçamento** (`QuotePage`), **Estoque** (`StockPage`).
- Jogos: One Piece, Pokémon, Riftbound, Lorcana, Gundam (game switcher + acento por jogo).

## Design tokens (exatos — replicar de `globals.css`)

| Token | Hex | Uso |
|---|---|---|
| Rosa (primária) | `#F6559B` | header, botões primários, preços, aba ativa, acentos |
| Rosa-claro | `#FDC4E5` | botões secundários, badges, labels pixel no escuro |
| Azul | `#1355B3` | fundo hero, CTA secundário, barras |
| Ink / outline | `#0b0b0c` | toda borda e sombra dura; superfície mais escura |
| Fundo | `#141416` | canvas do app |
| Surface | `#1f1f22` | cards, painéis, inputs |
| Texto sec / terc / on-soft | `#9a9aa2` / `#6f6f77` / `#8a3f68` | metas, fine print, texto em rosa-claro |

- **Semânticas de dado (MANTER distinguíveis):** ganho/verde, perda/vermelho, live/âmbar —
  integrar ao arcade dando a elas a borda ink, **sem virar tudo rosa/azul**.

## Estilo "arcade sticker" (regra central)

Superfície = cor sólida + borda `4–5px solid #0b0b0c` + radius 10–18px + sombra dura
`Xpx Ypx 0 #0b0b0c` (**sem blur**). Hover `arcade-press` (desliza 2px pra sombra e encolhe;
achata no `:active`). Chips pequenos: borda 3px, sem sombra.

- **Fontes:** Baloo 2 (títulos/H, peso 800) · Press Start 2P (labels/KPI captions/abas/badges —
  **SEM ACENTO**, Press Start 2P não tem glifos acentuados) · DM Sans (corpo).
- **Efeitos:** scanlines, `blink`, `bob`, `rise` — sempre respeitando `prefers-reduced-motion`.
- **Focus rings** nunca removidos (outline rosa-claro 3px, offset 2px).

## Adaptações pra um DASHBOARD (não é a vitrine marketing — é denso de dados)

- **Press Start 2P SÓ em rótulos curtos** (KPI caption, aba, badge, eyebrow de seção).
  NUNCA em números, células de tabela ou texto longo — números em DM Sans/mono com `tabular-nums`.
- **Sombra dura com parcimônia:** tratamento sticker completo em KPI cards, botões primários,
  game switcher e cabeçalhos de painel. Tabelas/linhas usam borda fina (2–3px) e sombra baixa/zero
  pra não virar ruído visual.
- **Gráficos** (histórico de preço, sparklines, movers): usar o estilo de barra arcade da vitrine
  (borda ink, fill chapado, rosa no ponto mais recente) — ver `SingleDetailView`.
- **Scanlines:** sutil ou desligado por padrão numa ferramenta de trabalho (opcional/toggle).
- **Manter o shell** sidebar+topbar (reskin, não reescrever a navegação) — a menos que a
  decisão inicial seja migrar pra uma tab bar arcade.
- **Acessibilidade:** evitar texto `#F6559B` sobre `#1355B3`; manter focus rings; contraste em dados.

## Abordagem de trabalho

1. Entrar em **PLAN MODE**, explorar `web/src`, e fazer perguntas ANTES de codar. Ex.:
   - Começar por quais views? (sugestão: shell + Deals primeiro, depois Portfolio/Tracking)
   - Manter shell sidebar ou virar **tab bar arcade** como a vitrine?
   - Quão longe levar o arcade nas tabelas densas (`DealsTable`, holdings)?
   - Centralizar tokens compartilhados com a vitrine (ambas são Tailwind v4 `@theme`)?
2. É **reskin**: PRESERVAR todos os dados, chamadas de API e lógica — só a camada visual muda.
3. Reaproveitar os mesmos tokens/helpers da vitrine (idealmente extrair pra um lugar comum
   pra não divergir).
4. **Verificar:** `npm run build` (tsc + vite) limpo + screenshots headless de cada view antes
   de dizer que terminou. Não commitar. Seguir `CLAUDE.md`/`lessons.md` (sem comentários no
   código, plan antes de codar, gate na aprovação, etc.).

**Comece lendo os arquivos de referência e a base do dashboard, depois apresente um plano
com perguntas — não escreva código ainda.**

---

### Contexto: como ficou a vitrine (pra alinhar expectativa)

A vitrine (`storefront/`, Next 15 + React 19 + Tailwind v4) foi reconstruída no estilo arcade a
partir de um handoff de design: 5 telas (Home, Browse singles, página da carta, selado, carrinho),
rotas reais, fontes self-hosted via `next/font`, catálogo de singles vindo do backend Go
(`GET /api/storefront`) com fallback mock. O dashboard deve herdar **a mesma linguagem visual**,
adaptada pra densidade de dados de um app de trabalho.
