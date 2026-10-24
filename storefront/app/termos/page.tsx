import type { Metadata } from "next";
import Link from "next/link";
import Container from "@/components/ui/Container";
import Section from "@/components/ui/DocSection";
import SectionHead from "@/components/ui/SectionHead";
import { STORE } from "@/lib/store";

export const metadata: Metadata = {
  title: "Termos de uso e venda",
  description: "Regras de compra na Collecta: preços, estoque, reserva no checkout, pagamento, condição das cartas e trocas.",
  alternates: { canonical: "/termos" },
};

export default function TermosPage() {
  return (
    <Container narrow className="py-9">
      <SectionHead title="Termos de uso e venda" eyebrow="COLLECTA" size="md" heading="h1" />
      <div className="grid gap-4">
        <Section title="A loja">
          <p>
            O site collectatcg.com.br é operado por {STORE.legalName}, CNPJ {STORE.cnpj}. Vendemos cartas avulsas,
            produto selado e acessórios de trading card games, com loja física em {STORE.city}-{STORE.state}.
          </p>
        </Section>
        <Section title="Preços e estoque">
          <p>
            Os preços estão em reais e podem mudar sem aviso; vale o preço do momento em que o pedido é pago. Boa parte
            do estoque é peça única, então o item só fica garantido depois do pagamento.
          </p>
          <p>
            Ao iniciar o pagamento, os itens ficam reservados por 30 minutos. Se o pagamento não for concluído nesse
            prazo, eles voltam para a loja.
          </p>
        </Section>
        <Section title="Condição das cartas">
          <p>
            Cada carta é conferida à mão e anunciada com a condição (NM, LP, MP, HP). Todas são originais. Produto
            selado é vendido com o lacre de fábrica.
          </p>
        </Section>
        <Section title="Pagamento">
          <p>
            Pix ou cartão em até 12x, processados pelo Mercado Pago. O pedido é confirmado quando o Mercado Pago aprova
            o pagamento.
          </p>
        </Section>
        <Section title="Entrega, trocas e devoluções">
          <p>
            Prazos e custos de envio estão na página de{" "}
            <Link href="/frete" className="text-brand-soft underline-offset-4 hover:underline">
              frete
            </Link>
            . Arrependimento, defeito e estorno seguem a{" "}
            <Link href="/politica-de-devolucao" className="text-brand-soft underline-offset-4 hover:underline">
              política de devolução
            </Link>{" "}
            e o Código de Defesa do Consumidor.
          </p>
        </Section>
        <Section title="Contato">
          <p>
            WhatsApp {STORE.phone}, e-mail {STORE.email}. Fica eleito o foro da comarca de {STORE.city}-{STORE.state},
            sem prejuízo do foro do domicílio do consumidor.
          </p>
        </Section>
      </div>
    </Container>
  );
}
