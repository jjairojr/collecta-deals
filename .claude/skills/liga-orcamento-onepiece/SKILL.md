---
name: liga-orcamento-onepiece
description: Confere uma lista de venda de cartas One Piece (colada de WhatsApp) contra o piso atual da LigaOnePiece usando os snapshots locais do tracking, e aponta o que está caro, no piso ou abaixo. Use quando o pedido for conferir/orçar uma lista de cartas One Piece, "essa lista bate com a liga?", avaliar proposta de vendedor ou lote de compra.
allowed-tools: Bash, Read, Write
---

# Orçamento One Piece — conferir lista de vendedor contra a Liga

Responde "essa lista bate com o site da Liga?" sem abrir o site: os snapshots
do tracking (`data/tracking/<SET>/*.json`, capturas a cada ~6h) guardam
**todas as ofertas** de cada carta com loja, condição, idioma, quantidade e
preço. Daí sai o piso NM, a comparação carta a carta e os destaques de
negociação.

Esta skill é **só análise — não grava nada em lugar nenhum**. Precificar o
NOSSO estoque é `liga-precificar`; auditar a loja é `liga-conferir`; publicar
cartas é `liga-singles-onepiece`. A versão Pokémon é `liga-orcamento-pokemon`
— o método é o mesmo, o que muda está neste arquivo.

---

## Fase 0 — Parsear a lista colada

```
1 Shanks (OP01-120) 350,00 cada
2 Boa Hancock (OP07-051 AA) 89,90
1 Luffy (ST01-001) 60,00 japones
```

- `<qtd> <nome> (<número>[ <variante>]) <preço> [cada] [idioma]`. Vírgula
  decimal. **Sem flag de idioma = assumir EN** — o mercado de One Piece na
  Liga é inglês por definição (`FloorLangs` do jogo é só `2`); PT e JP são
  exceção e a lista normalmente avisa.
- A variante pode vir por extenso ("alternate", "parallel", "manga") ou como
  sufixo. No snapshot ela faz parte do próprio número: `OP07-051-AA`.
  Sufixos comuns: `-AA` (Alternate Art), `-PA`/`-PAR` (Parallel), `-MA`
  (Manga), `-SP`, `-G` (Gold), `-FA` (Full Art), `-RE`, `-TR`, `-DP`
  (Double Pack), `-p1`.. (printing). Linha sem variante = número puro.
- Linha sem preço ou sem número → "não parseado" no relatório, nunca
  descartar em silêncio.

---

## Fase 1 — Resolver o diretório do set

O número já carrega o set (`OP16-118` → OP-16): números de One Piece são
globalmente únicos. Mas os **nomes dos diretórios não seguem uma regra só**:

- `OP-01`..`OP-16` (com hífen) · `EB01`..`EB03` (sem) · `ST-01`..`ST-13`
  (com) e `ST14`..`ST30` (sem) · `PRB`, `PRB2`. Teste os dois formatos
  (`glob` do Python NÃO faz brace expansion `{a,b}` — isso é do shell):
  `glob.glob('data/tracking/ST-09/*.json') or glob.glob('data/tracking/ST09/*.json')`.
- **Cartas EB podem morar dentro de edições OP.** EB04 não tem diretório
  próprio: seus cards estão nas edições OP-14/OP-15 da Liga (é assim que a
  Liga cataloga). Se o número "não existe" no diretório esperado, procure-o
  nos diretórios vizinhos por data antes de concluir ausência.
- **DON!! Cards reiniciam a numeração por edição** (`DON-001` é Mihawk em
  OP-14 e Luffy em OP-15; PRB2 tem a própria série com `-G`/`-DP`). Sem o
  contexto da edição na lista, um DON é inconferível — pergunte, não chute.

---

## Fase 2 — Casar cada carta no snapshot

Snapshot mais recente: `sorted(glob.glob('data/tracking/<SET>/*.json'))[-1]`;
reporte o `capturedAt`. O campo `number` traz set+número+variante
(`EB01-001-AA`) — normalize o hífen da lista (`OP16 118`, `OP16-118` e
`OP16118` são a mesma carta) e anexe o sufixo da variante.

- **Trava de nome (lição CEL/CCC).** Número casou ≠ carta certa. O `name` do
  snapshot (`Kouzuki Oden (Alternate Art) (EB01-001-AA)`) tem que conter o
  personagem da lista E a variante esperada. Nome que não bater → linha
  `⚠️ verificar`, preço fora das somas.
- Número ausente → **"sem dados"** (cobertura é parcial: o tracking segue uma
  lista curada por set). Nunca use o preço da versão normal para uma AA nem
  vice-versa — no snapshot elas são cartas distintas, com pisos que diferem
  10–50×.

---

## Fase 3 — Calcular os pisos

> ⚠️ **`lowBRL` é o piso cru de QUALQUER condição/idioma — nunca use como
> referência principal.** Recalcule de `stores[]` filtrando
> `priceKnown && known && quantity > 0`.

| Piso | Filtro | Uso |
| --- | --- | --- |
| NM-EN | `condition ∈ {1,2}` e `language == "2"` | **referência default** |
| NM no idioma da flag | `language == "6"` (JP) / `"8"` (PT) | quando a linha declara |
| menor do site | nenhum (= `lowBRL`) | coluna informativa |

Códigos: condição `1` M · `2` NM · `3` SP · `4` MP · `5` HP · `6` D; idioma
`2` EN · `6` JP · `8` PT · `11` PT/EN. Ofertas JP existem em starter antigo e
costumam estar **abaixo** do EN — se o menor do site vier de JP, isso não é
o piso do mercado EN.

O snippet canônico é o de `liga-orcamento-pokemon` Fase 3, trocando o
diretório para `data/tracking/` e o default de idioma para `{'2'}`.

---

## Fase 4 — Relatório (o que sempre entregar)

- **Veredito primeiro**: bate ou não, e o % agregado lista vs piso NM-EN.
- Tabela carta a carta: número, nome do snapshot, preço da lista, menor do
  site, piso NM-EN, diff %.
- Destaques: abaixo do piso (oportunidade) e >15% acima (negociação).
- Ressalvas: data/hora do snapshot; comparação assume NM; "sem dados" e
  "não parseado" com o valor que ficou de fora; DONs sem edição declarada.

---

## Variante B — orçamento feito como Coleção no próprio site

Link `ligaonepiece.com.br/?view=colecao/colecao&id=NNNN`: os preços já são o
menor preço da Liga por variante+qualidade. Mesma mecânica documentada em
`liga-orcamento-pokemon` (Variante B): abrir no Chrome do usuário (CF passa
sozinho em ~8s; ritmo 7–10s entre navegações, máx. 2 páginas por batch),
paginar com `&page=2`, conferir que o parse fecha com o total do site ao
centavo, e cruzar as cartas de maior valor com o piso NM-EN do snapshot.

---

## Armadilhas que já custaram análise errada

- **`lowBRL` ≠ piso NM-EN.** Cópia SP/JP barata na frente distorce o veredito.
- **EB04 mora dentro de OP-14/OP-15**; "não existe no diretório" não é "não
  existe no catálogo".
- **DON!! reinicia por edição** — sem edição, sem conferência.
- **Variante é parte da identidade da carta** (`-AA`/`-PA`/`-MA`...), não um
  detalhe: piso da normal e da AA diferem em ordem de grandeza.
- **Cobertura parcial não é ausência de mercado.** "Sem dados" = tracking não
  segue a carta, não que ela não tem oferta na Liga.

---

**Aprendeu algo novo? Acrescente aqui — este arquivo é o repositório desse
conhecimento.**
