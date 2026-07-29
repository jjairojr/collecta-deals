import type { Metadata } from "next";
import Container from "@/components/ui/Container";
import SectionHead from "@/components/ui/SectionHead";
import WhatsAppLink from "@/components/chrome/WhatsAppLink";

export const metadata: Metadata = {
  title: "Política de devolução e trocas",
  description:
    "Como funcionam devoluções, trocas e estornos na Collecta: 7 dias de arrependimento pelo CDC, produto com defeito ou diferente do anunciado, e como solicitar pelo WhatsApp.",
  alternates: { canonical: "/politica-de-devolucao" },
};

function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="sticker rounded-[14px] bg-surface p-6 sm:p-7 [--sh:6px]">
      <h2 className="font-display text-2xl font-bold leading-tight text-white">
        {title}
      </h2>
      <div className="mt-3 space-y-3 text-sm leading-relaxed text-[#c9c9d1]">
        {children}
      </div>
    </section>
  );
}

export default function PoliticaDeDevolucaoPage() {
  const whatsapp = process.env.NEXT_PUBLIC_WHATSAPP;
  const whatsappHref = whatsapp ? `https://wa.me/${whatsapp}` : "https://wa.me/";

  return (
    <Container narrow className="py-9">
      <SectionHead
        title="Política de devolução e trocas"
        eyebrow="COLLECTA"
        size="md"
        heading="h1"
      />

      <div className="grid gap-4">
        <Section title="7 dias para desistir da compra">
          <p>
            Você pode desistir da compra em até <strong>7 dias corridos</strong>{" "}
            contados a partir do recebimento do pedido, sem precisar justificar
            o motivo. É o direito de arrependimento previsto no artigo 49 do
            Código de Defesa do Consumidor, e vale para toda compra feita fora
            de loja física.
          </p>
          <p>
            Nesse caso devolvemos o valor total pago, incluindo o frete que você
            pagou na compra. O custo do frete de retorno é por nossa conta.
          </p>
        </Section>

        <Section title="Produto com defeito ou diferente do anunciado">
          <p>
            Se a carta chegar com dano que não estava descrito, em estado de
            conservação pior do que o anunciado, ou se enviarmos um item
            diferente do que você pediu, avise a gente em até{" "}
            <strong>90 dias</strong> do recebimento (prazo do artigo 26 do CDC
            para produtos duráveis).
          </p>
          <p>
            Você escolhe entre substituição por um item equivalente, devolução
            integral do valor pago ou abatimento proporcional do preço. Como boa
            parte do nosso estoque é peça única, nem sempre existe um item
            idêntico para troca — nesses casos o estorno integral fica
            disponível de imediato. Frete de retorno por nossa conta.
          </p>
        </Section>

        <Section title="Como solicitar">
          <p>
            Fale com a gente pelo WhatsApp com o número do pedido e, se for o
            caso, fotos da carta e da embalagem como chegaram. A gente confirma
            a solicitação e envia as instruções de postagem.
          </p>
          <p>
            <WhatsAppLink
              href={whatsappHref}
              origin="politica_devolucao"
              className="inline-flex min-h-11 items-center font-pixel text-[10px] text-white"
            >
              <span className="border-b-[3px] border-brand pb-1">
                FALAR NO WHATSAPP ›
              </span>
            </WhatsAppLink>
          </p>
        </Section>

        <Section title="Condições para a devolução">
          <p>
            A carta precisa voltar no mesmo estado em que foi enviada e com a
            mesma proteção que recebeu (sleeve, toploader ou case). Cartas
            devolvidas com dano causado após a entrega — vinco, risco, canto
            amassado, sleeve removida — não podem ser aceitas, porque o estado
            de conservação é justamente o que define o preço de uma carta.
          </p>
          <p>
            <strong>
              Produto selado só pode ser devolvido com o lacre original intacto.
            </strong>{" "}
            Booster box, blister, deck ou pacote que tenha sido aberto não é
            elegível para devolução ou troca — inclusive dentro dos 7 dias de
            arrependimento. Uma vez aberto, o conteúdo já foi revelado e o
            produto anunciado deixou de existir: o que voltaria não é o mesmo
            item que você comprou.
          </p>
          <p>
            Recomendamos postar com embalagem rígida e código de rastreio: até a
            devolução chegar até nós, ela ainda está sob sua guarda.
          </p>
        </Section>

        <Section title="Estorno">
          <p>
            Assim que a devolução chegar e for conferida, o estorno é feito pelo
            mesmo meio de pagamento usado na compra. O prazo depende do meio:
            devoluções instantâneas saem no mesmo dia útil, e as processadas por
            operadora de cartão seguem o calendário da própria operadora,
            podendo aparecer só na fatura seguinte.
          </p>
        </Section>
      </div>

      <p className="mt-6 text-xs leading-relaxed text-faint">
        Esta política complementa o Código de Defesa do Consumidor (Lei
        8.078/1990) e nunca o substitui — em qualquer conflito, prevalece o que
        for mais favorável a você.
      </p>
    </Container>
  );
}
