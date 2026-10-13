import type { Metadata } from "next";
import Image from "next/image";
import Link from "next/link";
import WhatsAppLink from "@/components/chrome/WhatsAppLink";
import JsonLd from "@/components/seo/JsonLd";
import Container from "@/components/ui/Container";
import SectionHead from "@/components/ui/SectionHead";
import { breadcrumbJsonLd } from "@/lib/seo";
import { STORE, STORE_ADDRESS_LINE, STORE_MAPS_EMBED_URL, STORE_MAPS_URL, STORE_PHOTOS, STORE_WHATSAPP_URL } from "@/lib/store";

export const metadata: Metadata = {
  title: "Loja de cartas Pokémon, One Piece e TCG em Goiânia",
  description:
    "Visite a Collecta TCG na Galeria Badra, Setor Oeste, Goiânia: singles, produto selado e acessórios de Pokémon, One Piece, Riftbound, Lorcana e Gundam, mesas para jogar e compra de coleções. Aberta todos os dias das 12h às 22h.",
  alternates: { canonical: "/loja" },
  openGraph: {
    title: "Collecta TCG — loja de cartas em Goiânia",
    description: "Galeria Badra, Setor Oeste. Todos os dias das 12h às 22h. Retire seu pedido na loja ou jogue com a gente.",
    url: "/loja",
    type: "website",
    siteName: "Collecta",
    locale: "pt_BR",
    images: [STORE_PHOTOS[0].src],
  },
};

const GAMES = ["Pokémon", "One Piece", "Riftbound", "Lorcana", "Gundam"];

function Panel({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="sticker rounded-[14px] bg-surface p-6 sm:p-7 [--sh:6px]">
      <h2 className="font-display text-2xl font-bold leading-tight text-white">{title}</h2>
      <div className="mt-3 space-y-3 text-sm leading-relaxed text-[#c9c9d1]">{children}</div>
    </section>
  );
}

export default function LojaPage() {
  return (
    <Container narrow className="py-9">
      <JsonLd
        data={breadcrumbJsonLd([
          { name: "Collecta", path: "/" },
          { name: "Loja física em Goiânia" },
        ])}
      />
      <SectionHead
        title="Loja de cartas TCG em Goiânia"
        eyebrow="COLLECTA · GALERIA BADRA · SETOR OESTE"
        size="md"
        heading="h1"
      />

      <div className="grid gap-4 lg:grid-cols-[1.1fr_.9fr] lg:items-start">
        <div className="grid gap-4">
          <Panel title="Onde fica">
            <address className="not-italic">
              <p className="text-white">{STORE_ADDRESS_LINE}</p>
              <p className="mt-1">Dentro da Galeria Badra, Loja 03, no Setor Oeste.</p>
            </address>
            <div className="flex flex-wrap gap-3 pt-1">
              <a
                href={STORE_MAPS_URL}
                target="_blank"
                rel="noopener"
                className="arcade-press sticker rounded-[10px] bg-brand px-4 py-2.5 font-pixel text-[9px] text-white [--sh:4px]"
              >
                COMO CHEGAR
              </a>
              <WhatsAppLink
                href={STORE_WHATSAPP_URL}
                origin="loja"
                className="arcade-press sticker rounded-[10px] bg-brand-soft px-4 py-2.5 font-pixel text-[9px] text-outline [--sh:4px]"
              >
                WHATSAPP
              </WhatsAppLink>
              <a
                href={STORE.instagram}
                target="_blank"
                rel="noopener"
                className="arcade-press sticker rounded-[10px] bg-royal px-4 py-2.5 font-pixel text-[9px] text-white [--sh:4px]"
              >
                INSTAGRAM
              </a>
            </div>
          </Panel>

          <Panel title="Horário de funcionamento">
            <table className="w-full text-sm">
              <tbody>
                {["Segunda", "Terça", "Quarta", "Quinta", "Sexta", "Sábado", "Domingo"].map((day) => (
                  <tr key={day} className="border-b border-outline/60 last:border-0">
                    <th scope="row" className="py-1.5 text-left font-normal text-[#c9c9d1]">{day}</th>
                    <td className="py-1.5 text-right font-pixel text-[9px] text-white">
                      {STORE.opens} – {STORE.closes}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
            <p>Aberta todos os dias. Telefone e WhatsApp: {STORE.phone}.</p>
          </Panel>

          <Panel title="O que você encontra">
            <p>
              Singles conferidas carta por carta, produto selado lacrado (boosters, booster boxes, decks e bundles) e
              acessórios como sleeves, deckboxes, toploaders e playmats de {GAMES.slice(0, -1).join(", ")} e {GAMES.at(-1)}.
            </p>
            <p>
              Compramos cartas e coleções, e a loja tem mesas para jogar. Pedidos feitos no site podem ser retirados
              em mãos sem frete.
            </p>
            <div className="flex flex-wrap gap-3 pt-1">
              <Link href="/singles" className="arcade-press sticker rounded-[10px] bg-royal px-4 py-2.5 font-pixel text-[9px] text-white [--sh:4px]">
                VER SINGLES
              </Link>
              <Link href="/selado" className="arcade-press sticker rounded-[10px] bg-royal px-4 py-2.5 font-pixel text-[9px] text-white [--sh:4px]">
                VER SELADOS
              </Link>
              <Link href="/acessorios" className="arcade-press sticker rounded-[10px] bg-royal px-4 py-2.5 font-pixel text-[9px] text-white [--sh:4px]">
                ACESSORIOS
              </Link>
            </div>
          </Panel>
        </div>

        <div className="grid gap-4">
          <div className="sticker overflow-hidden rounded-[14px] bg-outline [--sh:6px]">
            <iframe
              title="Mapa da Collecta TCG na Galeria Badra, Goiânia"
              src={STORE_MAPS_EMBED_URL}
              loading="lazy"
              referrerPolicy="no-referrer-when-downgrade"
              className="block aspect-[4/3] w-full border-0"
            />
          </div>
          <div className="grid grid-cols-3 gap-3">
            {STORE_PHOTOS.map((photo) => (
              <Image
                key={photo.src}
                src={photo.src}
                alt={photo.alt}
                width={photo.width}
                height={photo.height}
                className="sticker aspect-[4/5] w-full rounded-[10px] object-cover [--sh:4px]"
              />
            ))}
          </div>
          <p className="font-pixel text-[7px] leading-relaxed text-faint">
            PROJETO DA LOJA. FOTOS REAIS EM BREVE.
          </p>
        </div>
      </div>
    </Container>
  );
}
