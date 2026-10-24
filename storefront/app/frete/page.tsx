import type { Metadata } from "next";
import Link from "next/link";
import Container from "@/components/ui/Container";
import Section from "@/components/ui/DocSection";
import SectionHead from "@/components/ui/SectionHead";
import { brl } from "@/lib/format";
import { SHIPPING, STORE_ADDRESS_LINE } from "@/lib/store";

const handling = `${SHIPPING.handlingDays[1]} ${SHIPPING.handlingDays[1] === 1 ? "dia útil" : "dias úteis"}`;

export const metadata: Metadata = {
  title: "Frete e prazo de entrega",
  description: `Como a Collecta envia cartas e produtos selados: frete calculado pelo CEP, postagem em até ${handling}, frete grátis acima de ${brl(SHIPPING.freeAboveBRL * 100)} e retirada na loja em Goiânia.`,
  alternates: { canonical: "/frete" },
};

export default function FretePage() {
  return (
    <Container narrow className="py-9">
      <SectionHead title="Frete e prazo de entrega" eyebrow="COLLECTA" size="md" heading="h1" />
      <div className="grid gap-4">
        <Section title="Quanto custa">
          <p>
            O frete é calculado pelo CEP no checkout, com as opções de entrega disponíveis para o seu endereço e o preço
            de cada uma. O envio mais barato começa em cerca de {brl(SHIPPING.minBRL * 100)}.
          </p>
          <p>
            <strong>Frete grátis acima de {brl(SHIPPING.freeAboveBRL * 100)}</strong> em produtos: a opção de envio mais
            barata sai de graça.
          </p>
        </Section>
        <Section title="Prazo">
          <p>
            Postamos o pedido em até {handling} depois da confirmação do pagamento. O prazo
            de transporte depende da opção escolhida e do CEP, normalmente de {SHIPPING.transitDays[0]} a{" "}
            {SHIPPING.transitDays[1]} dias úteis, e aparece no checkout antes de você pagar.
          </p>
          <p>Todo envio tem código de rastreio, que mandamos assim que o pedido é postado.</p>
        </Section>
        <Section title="Embalagem">
          <p>
            Cada carta vai em sleeve e toploader, dentro de embalagem rígida. Produto selado segue com proteção extra
            para não amassar a caixa nem o lacre.
          </p>
        </Section>
        <Section title="Retirada na loja">
          <p>
            Em Goiânia você pode escolher retirar na loja no checkout, sem frete. Depois do pagamento a gente combina o
            dia pelo WhatsApp. Endereço: {STORE_ADDRESS_LINE}.
          </p>
        </Section>
        <Section title="Problema na entrega">
          <p>
            Se o pedido chegar danificado ou diferente do anunciado, veja a{" "}
            <Link href="/politica-de-devolucao" className="text-brand-soft underline-offset-4 hover:underline">
              política de devolução e trocas
            </Link>
            .
          </p>
        </Section>
      </div>
    </Container>
  );
}
