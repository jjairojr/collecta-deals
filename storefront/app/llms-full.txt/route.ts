import { loadCatalog } from "@/lib/catalog";
import { brl } from "@/lib/format";
import { freeShippingMinLabel } from "@/lib/free-shipping";
import { GAME_NAMES, isGameId } from "@/lib/games";
import { GUIDES } from "@/lib/guides";
import { llmsLines } from "@/lib/llms";
import { MAX_INSTALLMENTS, SEM_JUROS } from "@/lib/payment";
import { SEALED_GENERAL } from "@/lib/sealed-faq";
import { setsOf } from "@/lib/sets";
import { absoluteURL } from "@/lib/site";
import { SHIPPING, STORE, STORE_ADDRESS_LINE } from "@/lib/store";

// Versão longa do llms.txt: o mesmo índice, mais as políticas em texto, as
// perguntas frequentes e um resumo das coleções de cartas avulsas (contagem e
// faixa de preço), sem listar carta por carta.
export const revalidate = 300;

export async function GET(): Promise<Response> {
  const catalog = await loadCatalog();
  const { singles, games, freeShippingMinBRL } = catalog;
  const handling = `${SHIPPING.handlingDays[1]} ${SHIPPING.handlingDays[1] === 1 ? "dia útil" : "dias úteis"}`;
  const lines = llmsLines(catalog);

  lines.push(
    "",
    "## Frete e entrega",
    `O frete é calculado pelo CEP no checkout, com as opções de entrega disponíveis para o endereço e o preço de cada uma. O envio mais barato começa em cerca de ${brl(SHIPPING.minBRL * 100)}.`,
    ...(freeShippingMinBRL > 0
      ? [
          `Frete grátis no PAC para todo o Brasil em compras a partir de ${freeShippingMinLabel(freeShippingMinBRL)} em produtos: no checkout, a opção PAC dos Correios sai de graça e as outras opções continuam pagas. O valor conta o preço da forma de pagamento escolhida (Pix ou cartão), sem o frete. Se o PAC não atender o CEP, não há opção grátis.`,
        ]
      : []),
    `A loja posta o pedido em até ${handling} depois da confirmação do pagamento. O transporte leva normalmente de ${SHIPPING.transitDays[0]} a ${SHIPPING.transitDays[1]} dias úteis, conforme a opção e o CEP, e o prazo aparece no checkout antes do pagamento. Todo envio tem código de rastreio.`,
    "Cada carta vai em sleeve e toploader, dentro de embalagem rígida; produto selado segue com proteção extra para não amassar a caixa nem o lacre.",
    `Em Goiânia dá para retirar na loja, sem frete: ${STORE_ADDRESS_LINE}. Depois do pagamento, o dia é combinado pelo WhatsApp.`,
    `Mais: ${absoluteURL("/frete")}`,
    "",
    "## Pagamento",
    `Pix ou cartão pelo Mercado Pago, direto no checkout do site. No cartão, em até ${MAX_INSTALLMENTS}x${SEM_JUROS}, sobre o preço no cartão. O pedido é confirmado quando o Mercado Pago aprova o pagamento. Ao iniciar o pagamento, os itens ficam reservados por 2 horas.`,
    "",
    "## Devolução e trocas",
    "Arrependimento: até 7 dias corridos depois do recebimento, sem justificar o motivo (artigo 49 do CDC); a loja devolve o valor total pago, incluindo o frete da compra, e paga o frete de retorno.",
    "Defeito ou item diferente do anunciado: avisar em até 90 dias do recebimento (artigo 26 do CDC); o cliente escolhe entre substituição por item equivalente, devolução integral ou abatimento proporcional. Como boa parte do estoque é peça única, nem sempre há item idêntico para troca, e o estorno integral fica disponível na hora. Frete de retorno por conta da loja.",
    `Como pedir: pelo WhatsApp ${STORE.phone}, com o número do pedido e, se for o caso, fotos da carta e da embalagem como chegaram. A carta precisa voltar no mesmo estado e com a mesma proteção em que foi enviada.`,
    `Mais: ${absoluteURL("/politica-de-devolucao")}`,
    "",
    "## Termos de venda (resumo)",
    `Site operado por ${STORE.legalName}, CNPJ ${STORE.cnpj}. Preços em reais, que podem mudar sem aviso; vale o preço do momento em que o pedido é pago. Boa parte do estoque é peça única: o item só fica garantido depois do pagamento. Cada carta é conferida à mão e anunciada com a condição (NM, LP, MP, HP); todas são originais. Produto selado é vendido com o lacre de fábrica.`,
    `Mais: ${absoluteURL("/termos")}`,
    "",
    "## Loja física",
    `${STORE_ADDRESS_LINE}. Aberta todos os dias, das ${STORE.opens} às ${STORE.closes}. Telefone e WhatsApp: ${STORE.phone}. Torneios de card games toda semana (com premiação, gratuitos e 4fun) e mesas livres para jogar nos outros dias. A loja também compra cartas e coleções.`,
    `Mais: ${absoluteURL("/loja")}`,
    "",
    "## Cartas graduadas",
    "Cartas graduadas (slabs) são cartas avaliadas e lacradas por uma empresa de graduação, com a nota na etiqueta. A Collecta anuncia cada uma com a graduadora e a nota (PSA, BGS, CGC, SGC ou TAG) e foto real da slab.",
    "",
    "## Perguntas frequentes",
    ...SEALED_GENERAL.flatMap((f) => [`### ${f.q}`, f.a, ""]),
    ...GUIDES.flatMap((g) => g.faq.map((f) => `### ${f.q}\n${f.a}\n`)),
    "",
    "## Cartas avulsas por coleção",
  );
  for (const g of games.filter((x) => isGameId(x.id))) {
    const ofGame = singles.filter((s) => s.game === g.id);
    lines.push("", `### ${GAME_NAMES[g.id]} (${ofGame.length} cartas)`);
    for (const set of setsOf(ofGame, g.id)) {
      const prices = ofGame.filter((s) => s.set === set.code).map((s) => s.price);
      const max = Math.max(...prices);
      lines.push(
        `- [${set.label}](${absoluteURL(`/singles/${g.id}/colecao/${set.slug}`)}): ${set.count} ${set.count === 1 ? "carta" : "cartas"}, de ${brl(set.minPrice)}${max > set.minPrice ? ` a ${brl(max)}` : ""} no Pix`,
      );
    }
  }
  lines.push("", `Atualizado em ${new Date().toISOString().slice(0, 10)}.`, "");
  return new Response(lines.join("\n"), {
    headers: { "Content-Type": "text/markdown; charset=utf-8" },
  });
}
