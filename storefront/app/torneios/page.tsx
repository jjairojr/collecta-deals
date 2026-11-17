import type { Metadata } from "next";
import Link from "next/link";
import TrackedLink from "@/components/chrome/TrackedLink";
import WhatsAppLink from "@/components/chrome/WhatsAppLink";
import JsonLd from "@/components/seo/JsonLd";
import Container from "@/components/ui/Container";
import DocSection from "@/components/ui/DocSection";
import FaqSection from "@/components/ui/FaqSection";
import SectionHead from "@/components/ui/SectionHead";
import { CategoryIntro } from "@/components/singles/CategoryExtras";
import { breadcrumbJsonLd, type FaqItem } from "@/lib/seo";
import { STORE, STORE_ADDRESS_LINE, STORE_MAPS_URL, STORE_WHATSAPP_URL } from "@/lib/store";

export const metadata: Metadata = {
  title: "Torneios de TCG em Goiânia: semanais, com premiação e gratuitos",
  description:
    "A Collecta recebe torneios de card games toda semana em Goiânia: torneios com premiação, torneios gratuitos e encontros 4fun para jogar por diversão. Galeria Badra, Setor Oeste, com mesas para jogar todos os dias.",
  alternates: { canonical: "/torneios" },
};

const FAQ: FaqItem[] = [
  {
    q: "Onde tem torneio de TCG em Goiânia?",
    a: `Na Collecta TCG, na ${STORE_ADDRESS_LINE}. A loja recebe torneios de card games toda semana, com opções com premiação, gratuitas e 4fun.`,
  },
  {
    q: "Os torneios da Collecta são pagos?",
    a: "Há os dois tipos. A loja faz torneios com premiação, cuja inscrição é informada na agenda de cada evento, e também torneios gratuitos e encontros 4fun.",
  },
  {
    q: "O que é um torneio 4fun?",
    a: "É um encontro para jogar por diversão, sem pressão de resultado. Serve para testar deck, conhecer outros jogadores e aprender o jogo.",
  },
  {
    q: "Posso participar sendo iniciante?",
    a: "Sim. Os torneios gratuitos e os 4fun são a porta de entrada para quem está começando.",
  },
  {
    q: "Como fico sabendo dos próximos torneios?",
    a: `A agenda da semana sai no Instagram ${STORE.instagramHandle} e pelo WhatsApp da loja, ${STORE.phone}.`,
  },
  {
    q: "Dá para jogar na loja fora dos dias de torneio?",
    a: `Sim. A loja tem mesas para jogar e abre todos os dias, das ${STORE.opens} às ${STORE.closes}.`,
  },
];

const chip = "arcade-press sticker rounded-[10px] px-4 py-2.5 font-pixel text-[9px] [--sh:4px]";

export default function TorneiosPage() {
  return (
    <>
      <JsonLd data={breadcrumbJsonLd([{ name: "Collecta", path: "/" }, { name: "Torneios" }])} />
      <Container narrow className="py-9">
        <SectionHead
          title="Torneios de TCG em Goiânia"
          eyebrow="TODA SEMANA NA COLLECTA"
          size="md"
          heading="h1"
        />
        <CategoryIntro>
          A Collecta é loja e ponto de encontro de card games em Goiânia. Toda semana tem torneio na loja: torneios com
          premiação, torneios gratuitos e encontros 4fun, para quem quer jogar por diversão. Fica na Galeria Badra,
          Setor Oeste, com mesas para jogar todos os dias, das {STORE.opens} às {STORE.closes}.
        </CategoryIntro>
        <div className="mt-5 flex flex-wrap gap-3">
          <WhatsAppLink href={STORE_WHATSAPP_URL} origin="torneios" className={`${chip} bg-brand text-white`}>
            AGENDA NO WHATSAPP
          </WhatsAppLink>
          <TrackedLink
            href={STORE.instagram}
            track={{ kind: "social", network: "instagram", origin: "torneios" }}
            className={`${chip} bg-royal text-white`}
          >
            INSTAGRAM {STORE.instagramHandle.toUpperCase()}
          </TrackedLink>
          <TrackedLink
            href={STORE_MAPS_URL}
            track={{ kind: "directions", origin: "torneios" }}
            className={`${chip} bg-brand-soft text-outline`}
          >
            COMO CHEGAR
          </TrackedLink>
        </div>

        <div className="mt-8 grid gap-4 sm:grid-cols-3">
          <DocSection title="Com premiação">
            <p>Torneios competitivos, com prêmios para os melhores colocados.</p>
            <p>Para quem quer testar o deck valendo e subir de nível.</p>
          </DocSection>
          <DocSection title="Gratuitos">
            <p>Torneios sem taxa de inscrição: é só chegar com o deck e jogar.</p>
            <p>Boa entrada para quem nunca jogou um torneio.</p>
          </DocSection>
          <DocSection title="4fun">
            <p>Encontros para jogar por diversão, sem pressão de resultado.</p>
            <p>Bom para testar listas, conhecer a galera e aprender o jogo.</p>
          </DocSection>
        </div>

        <div className="mt-4 grid gap-4 sm:grid-cols-2">
          <DocSection title="Como participar">
            <p>
              A agenda de cada semana, com jogo, dia e horário, sai no Instagram {STORE.instagramHandle} e pelo WhatsApp
              da loja. Chame no WhatsApp para saber do próximo.
            </p>
            <p>Leve o seu deck. Sleeves, deck box e cartas avulsas para fechar a lista você encontra na própria loja.</p>
          </DocSection>
          <DocSection title="Onde fica">
            <p className="text-white">{STORE_ADDRESS_LINE}</p>
            <p>
              Aberta todos os dias, das {STORE.opens} às {STORE.closes}. Veja mapa e fotos na página da{" "}
              <Link href="/loja" className="text-brand-soft underline-offset-4 hover:underline">
                loja física
              </Link>
              .
            </p>
          </DocSection>
        </div>

        <FaqSection items={FAQ} />

        <p className="mt-8 flex flex-wrap gap-x-4 gap-y-2 text-sm text-muted">
          {[
            { href: "/pokemon-tcg", label: "Pokémon TCG" },
            { href: "/one-piece-card-game", label: "One Piece Card Game" },
            { href: "/riftbound", label: "Riftbound" },
            { href: "/acessorios", label: "Acessórios" },
            { href: "/guia", label: "Guias para começar" },
          ].map((l) => (
            <Link key={l.href} href={l.href} className="text-brand-soft underline-offset-4 hover:underline">
              {l.label}
            </Link>
          ))}
        </p>
      </Container>
    </>
  );
}
