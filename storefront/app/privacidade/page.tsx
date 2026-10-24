import type { Metadata } from "next";
import Container from "@/components/ui/Container";
import Section from "@/components/ui/DocSection";
import SectionHead from "@/components/ui/SectionHead";
import { STORE } from "@/lib/store";

export const metadata: Metadata = {
  title: "Política de privacidade",
  description:
    "Quais dados a Collecta coleta, para que usa, com quem compartilha e como exercer seus direitos pela LGPD.",
  alternates: { canonical: "/privacidade" },
};

export default function PrivacidadePage() {
  return (
    <Container narrow className="py-9">
      <SectionHead title="Política de privacidade" eyebrow="COLLECTA" size="md" heading="h1" />
      <div className="grid gap-4">
        <Section title="Quem trata os seus dados">
          <p>
            {STORE.legalName}, CNPJ {STORE.cnpj}, com loja na {STORE.street}, {STORE.building}, {STORE.city}-
            {STORE.state}. Para qualquer assunto de privacidade, fale pelo e-mail {STORE.email} ou pelo WhatsApp{" "}
            {STORE.phone}.
          </p>
        </Section>
        <Section title="O que coletamos">
          <p>
            Para processar um pedido: nome, e-mail, telefone, CPF e endereço de entrega. Se você cria uma conta, também
            guardamos o login e o histórico dos seus pedidos e cupons.
          </p>
          <p>
            Ao navegar, usamos cookies do Google Analytics e do Meta Pixel para medir visitas e o desempenho dos
            anúncios. Eles registram páginas vistas e produtos adicionados ao carrinho, sem dados de pagamento.
          </p>
        </Section>
        <Section title="Para que usamos">
          <p>
            Para entregar o pedido, falar com você sobre a compra, cumprir obrigações legais
            e melhorar a loja. Não vendemos seus dados.
          </p>
        </Section>
        <Section title="Com quem compartilhamos">
          <p>
            Com o Mercado Pago, que processa o pagamento (os dados do cartão ficam com eles, nunca com a gente), com a
            transportadora e a plataforma de frete que entregam o pedido, e com Google e Meta para as medições acima.
          </p>
        </Section>
        <Section title="Seus direitos">
          <p>
            Pela Lei Geral de Proteção de Dados (Lei 13.709/2018) você pode pedir acesso, correção, portabilidade ou
            exclusão dos seus dados, e revogar consentimentos. Mande o pedido para {STORE.email}. Dados de pedidos
            podem ser mantidos pelo prazo exigido pela legislação fiscal.
          </p>
        </Section>
      </div>
    </Container>
  );
}
