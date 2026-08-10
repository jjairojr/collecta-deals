---
name: liga-orcamento-riftbound
description: Confere uma lista de venda de cartas Riftbound (colada de WhatsApp) contra o piso atual da LigaRiftbound usando os snapshots locais do tracking, e aponta o que está caro, no piso ou abaixo. Use quando o pedido for conferir/orçar uma lista de cartas Riftbound, "essa lista bate com a liga?", avaliar proposta de vendedor ou lote de compra.
allowed-tools: Bash, Read, Write
---

# Orçamento Riftbound — conferir lista de vendedor contra a Liga

Responde "essa lista bate com o site da Liga?" sem abrir o site: os snapshots
do tracking (`data/tracking-rft/<SET>/*.json`, capturas a cada ~6h) guardam
**todas as ofertas** de cada carta com loja, condição, idioma, quantidade e
preço.

Esta skill é **só análise — não grava nada em lugar nenhum**. Publicar cartas
é `liga-singles-riftbound`; precificar o NOSSO estoque é `liga-precificar`.
A versão Pokémon é `liga-orcamento-pokemon` — método idêntico; o que muda
está neste arquivo. Existe também `cmd/sellerquote` (Go), mas ele é para
páginas salvas do MyP Cards, não para lista colada.

---

## Fase 0 — Parsear a lista colada

```
1 Jinx - Rebel (202) 38,99 cada
1 Master Yi - Tempered (113A) 35,00 Origins
```

- `<qtd> <nome> (<número>) <preço> [cada] [set/idioma]`. Vírgula decimal.
  **Sem flag de idioma = assumir EN** — o piso do jogo é inglês
  (`FloorLangs: ["2"]`); o mercado BR carrega chinês (idioma `10`) bem mais
  barato, que NÃO é referência.
- O número é cru e pode ter sufixo de variante colado: `113A` (Alternate
  Art), `308S`-style para showcase. O sufixo é parte da identidade.
- Linha sem preço ou sem número → "não parseado" no relatório.

---

## Fase 1 — Resolver o set (obrigatório — número repete entre sets)

Números de Riftbound **não são globais**: `106` existe em UNL e em OGN. O set
tem que vir do contexto da lista (cabeçalho, nome do set por extenso) ou do
desempate por nome.

Diretórios em `data/tracking-rft/`: `OGN` (Origins) · `OGN-PR` (promos de
Origins) · `OGS` · `SFD` (Spiritforged; MyP chama de SPF) · `ROPP` (Proving
Grounds; MyP chama de OPP) · `UNL` (set atual). Liste com `ls` — sets novos
aparecem sozinhos.

Sem set declarado: procure o número em TODOS os diretórios e desempate pelo
NOME da carta (formato `Personagem - Título (num)`). Se mais de um set casar
número E nome, pare e pergunte — nunca chute.

---

## Fase 2 — Casar cada carta no snapshot

Snapshot mais recente: `sorted(glob.glob('data/tracking-rft/<SET>/*.json'))[-1]`;
reporte o `capturedAt`. O campo `number` vem cru com sufixo (`'113A'`), sem
zero-padding consistente (`'10'`, `'106'`) — compare como string exata e, se
falhar, sem/com zeros à esquerda.

- **Trava de nome.** `name` do snapshot é
  `Master Yi - Tempered (Alternate Art) (113A)` — o personagem E o título têm
  que bater com a lista. Nome divergente → `⚠️ verificar`, fora das somas.
- Número ausente → **"sem dados"** (cobertura parcial; OGN-PR tem só 9
  cartas, por exemplo). Nunca use o piso da versão sem sufixo para uma `A`.

---

## Fase 3 — Calcular os pisos

> ⚠️ **`lowBRL` é o piso cru de QUALQUER condição/idioma.** Em Riftbound o
> caso clássico é oferta em **chinês (idioma `10`)** na frente do EN — o
> jogo saiu primeiro na China e a cópia CN vale menos. Nunca use como
> referência principal.

Recalcule de `stores[]` com `priceKnown && known && quantity > 0`:

| Piso | Filtro | Uso |
| --- | --- | --- |
| NM-EN | `condition ∈ {1,2}` e `language == "2"` | **referência default** |
| NM-CN | `language == "10"` | só se a lista declarar "chinês" |
| menor do site | nenhum (= `lowBRL`) | coluna informativa |

Códigos: condição `1` M · `2` NM · `3` SP · `4` MP · `5` HP · `6` D; idioma
`2` EN · `10` ZH · `8` PT · `6` JP.

O snippet canônico é o de `liga-orcamento-pokemon` Fase 3, trocando o
diretório para `data/tracking-rft/` e o default de idioma para `{'2'}`.

---

## Fase 4 — Relatório (o que sempre entregar)

- **Veredito primeiro**: bate ou não, e o % agregado lista vs piso NM-EN.
- Tabela carta a carta: set, número, nome do snapshot, preço da lista, menor
  do site, piso NM-EN, diff %.
- Destaques: abaixo do piso e >15% acima.
- Ressalvas: data/hora do snapshot; comparação assume NM; se o "menor do
  site" de alguma carta vier de oferta CN, diga explicitamente; "sem dados"
  e "não parseado" com o valor de fora.

---

## Variante B — orçamento feito como Coleção no próprio site

Link `ligariftbound.com.br/?view=colecao/colecao&id=NNNN`: preços já são o
menor da Liga por variante+qualidade. Mesma mecânica documentada em
`liga-orcamento-pokemon` (Variante B): Chrome do usuário, CF passa sozinho em
~8s, ritmo 7–10s, `&page=2`, parse fecha ao centavo, cruzar as cartas caras
com o piso NM-EN do snapshot.

---

## Armadilhas que já custaram análise errada

- **Chinês na frente do piso.** O menor preço absoluto frequentemente é CN;
  comparar lista EN contra piso CN condena preço justo como "caro".
- **Número sem set é ambíguo** — sempre resolver o set antes, desempate por
  nome, nunca por chute.
- **Sufixo do número é a variante** (`113A` ≠ `113`); pisos diferem muito.
- **Cobertura parcial não é ausência de mercado.**
- **`lowBRL` ≠ piso NM-EN.**

---

**Aprendeu algo novo? Acrescente aqui — este arquivo é o repositório desse
conhecimento.**
