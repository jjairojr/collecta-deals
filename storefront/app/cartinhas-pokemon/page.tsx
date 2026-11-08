import Link from "next/link";
import type { Metadata } from "next";
import Container from "@/components/ui/Container";
import DocSection from "@/components/ui/DocSection";
import FaqSection from "@/components/ui/FaqSection";
import SectionHead from "@/components/ui/SectionHead";
import SealedGrid from "@/components/SealedGrid";
import { CategoryIntro } from "@/components/singles/CategoryExtras";
import { SealedCollectionLinks } from "@/components/sealed/SealedExtras";
import JsonLd from "@/components/seo/JsonLd";
import { loadCatalog } from "@/lib/catalog";
import { brl } from "@/lib/format";
import { liveCollections } from "@/lib/sealed-guide";
import { SEALED_SHIPPING } from "@/lib/sealed-faq";
import { breadcrumbJsonLd, itemListJsonLd, type FaqItem } from "@/lib/seo";
import { STORE_ADDRESS_LINE } from "@/lib/store";
import type { Sealed } from "@/lib/types";

export const revalidate = 300;

const PATH = "/cartinhas-pokemon";

function cheapest(list: Sealed[], match: RegExp): Sealed | undefined {
  return list.filter((s) => match.test(s.name)).sort((a, b) => a.price - b.price)[0];
}

async function pokemonShelf() {
  const { sealed, singles, live } = await loadCatalog();
  const list = sealed.filter((s) => s.game === "pokemon");
  const cards = singles.filter((s) => s.game === "pokemon");
  return {
    sealed,
    list,
    singles: cards.length,
    singlesFrom: cards.length > 0 ? Math.min(...cards.map((s) => s.price)) : undefined,
    blister: cheapest(list, /blister/i),
    box: cheapest(list, /caixa de booster|booster box/i),
    live,
  };
}

export async function generateMetadata(): Promise<Metadata> {
  const { blister, live } = await pokemonShelf();
  return {
    title: "Cartinhas Pokémon originais: booster, box, blister e cartas avulsas",
    description: `Onde comprar cartinhas Pokémon originais: ${blister ? `blister a partir de ${brl(blister.price)}, ` : ""}booster box, ETB e cartas avulsas em português, inglês, japonês e chinês. Loja física em Goiânia e envio para todo o Brasil.`,
    alternates: { canonical: PATH },
    robots: live ? undefined : { index: false },
  };
}

export default async function CartinhasPokemonPage() {
  const { sealed, list, singles, singlesFrom, blister, box } = await pokemonShelf();
  const shown = [...list].sort((a, b) => Number(Boolean(b.featured)) - Number(Boolean(a.featured))).slice(0, 8);
  const priceAnswer = [
    blister ? `o blister mais barato está por ${brl(blister.price)} (${blister.headline ?? blister.name})` : "",
    box ? `a booster box mais em conta por ${brl(box.price)} (${box.headline ?? box.name})` : "",
  ]
    .filter(Boolean)
    .join(" e ");
  const faq: FaqItem[] = [
    {
      q: "Onde comprar cartinhas Pokémon originais?",
      a: "Em lojas especializadas que vendem só produto oficial, como a Collecta: selados com lacre de fábrica e cartas avulsas conferidas uma a uma, com loja física em Goiânia e envio para todo o Brasil.",
    },
    ...(priceAnswer
      ? [{ q: "Quanto custam as cartinhas Pokémon?", a: `Na Collecta, ${priceAnswer}.${singlesFrom !== undefined ? ` Cartas avulsas saem a partir de ${brl(singlesFrom)} e variam com a raridade.` : ""}` }]
      : []),
    {
      q: "Qual a diferença entre booster, blister, ETB e booster box?",
      a: "O booster é o pacote de cartas. O blister junta alguns boosters com uma carta promocional. A ETB (Coleção Treinador Avançado) traz boosters, carta promocional, sleeves, energias, dados e marcadores. A booster box é a caixa display fechada com os boosters da coleção, o menor preço por pacote.",
    },
    {
      q: "Tem cartinhas Pokémon em português?",
      a: "Sim. As coleções atuais saem em português pela Copag, e também temos produtos em inglês, japonês e chinês para colecionadores.",
    },
    SEALED_SHIPPING,
  ];
  return (
    <>
      <JsonLd
        data={breadcrumbJsonLd([
          { name: "Collecta", path: "/" },
          { name: "Cartinhas Pokémon" },
        ])}
      />
      {shown.length > 0 && (
        <JsonLd data={itemListJsonLd("Cartinhas Pokémon seladas", shown.map((s) => `/selado/${s.slug}`))} />
      )}
      <Container className="py-9">
        <SectionHead title="Cartinhas Pokémon" eyebrow="ONDE COMPRAR ORIGINAIS" size="md" heading="h1" />
        <CategoryIntro>
          Na Collecta você compra cartinhas Pokémon originais: {list.length} produtos selados
          {blister ? `, com blister a partir de ${brl(blister.price)}` : ""}, booster box e ETB lacrados, e {singles}{" "}
          cartas avulsas conferidas uma a uma. Tem produto em português, inglês, japonês e chinês, loja física em Goiânia
          e envio para todo o Brasil.
        </CategoryIntro>

        <div className="mt-8 grid gap-4 sm:grid-cols-2">
          <DocSection title="Por onde começar">
            <p>
              <strong className="text-white">Para presentear ou começar a colecionar:</strong> um blister (alguns boosters
              com carta promocional) ou uma Coleção Treinador Avançado (ETB).
            </p>
            <p>
              <strong className="text-white">Para abrir em quantidade ou guardar lacrado:</strong> a booster box, que nas
              coleções principais traz 36 pacotes.
            </p>
            <p>
              <strong className="text-white">Para buscar uma carta específica:</strong> as{" "}
              <Link href="/singles/pokemon" className="text-brand-soft underline-offset-4 hover:underline">
                cartas Pokémon avulsas
              </Link>
              , com a condição descrita em cada anúncio.
            </p>
          </DocSection>
          <DocSection title="Como saber se a cartinha é original">
            <p>Compre de loja com CNPJ e endereço. A Collecta tem loja física: {STORE_ADDRESS_LINE}.</p>
            <p>
              No Brasil, os produtos em português são distribuídos pela Copag. Desconfie de caixa sem lacre de fábrica ou
              com plástico refeito.
            </p>
            <p>Preço muito abaixo do mercado é o sinal mais comum de falsificação ou de produto relacrado.</p>
          </DocSection>
        </div>

        <section className="mt-10">
          <SectionHead
            title="Selados Pokémon em estoque"
            eyebrow="BOOSTER BOX · ETB · BLISTER"
            size="md"
            action={{ label: "VER TODOS ›", href: "/selado/pokemon" }}
          />
          <SealedGrid list={shown} listId="cartinhas_pokemon" listName="Cartinhas Pokémon" />
        </section>

        <SealedCollectionLinks title="Coleções Pokémon" collections={liveCollections(sealed, "pokemon")} />
        <FaqSection items={faq} />
      </Container>
    </>
  );
}
