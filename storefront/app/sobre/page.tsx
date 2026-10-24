import type { Metadata } from "next";
import Link from "next/link";
import Container from "@/components/ui/Container";
import Section from "@/components/ui/DocSection";
import SectionHead from "@/components/ui/SectionHead";
import { STORE, STORE_ADDRESS_LINE } from "@/lib/store";

export const metadata: Metadata = {
  title: "Sobre a Collecta",
  description: `A Collecta é uma loja de cartas TCG em ${STORE.city}: Pokémon, One Piece, Riftbound e mais, com loja física e envio para todo o Brasil.`,
  alternates: { canonical: "/sobre" },
};

export default function SobrePage() {
  return (
    <Container narrow className="py-9">
      <SectionHead title="Sobre a Collecta" eyebrow="LOJA DE CARTAS TCG" size="md" heading="h1" />
      <div className="grid gap-4">
        <Section title="Quem somos">
          <p>
            A Collecta é uma loja de trading card games de {STORE.city}. Vendemos cartas avulsas, booster boxes, ETBs e
            acessórios de Pokémon, One Piece Card Game, Riftbound, Disney Lorcana e Gundam Card Game, para jogadores e
            colecionadores de todo o Brasil.
          </p>
          <p>
            Toda carta é original e conferida à mão quando entra no estoque e de novo antes do envio, com a condição
            descrita no anúncio.
          </p>
        </Section>
        <Section title="Loja física">
          <p>
            {STORE_ADDRESS_LINE}. Aberta todos os dias, das {STORE.opens} às {STORE.closes}.{" "}
            <Link href="/loja" className="text-brand-soft underline-offset-4 hover:underline">
              Ver mapa e fotos
            </Link>
            .
          </p>
        </Section>
        <Section title="Dados da empresa">
          <p>
            {STORE.legalName}, CNPJ {STORE.cnpj}. Contato: WhatsApp {STORE.phone}, e-mail {STORE.email}, Instagram{" "}
            {STORE.instagramHandle}.
          </p>
        </Section>
      </div>
    </Container>
  );
}
