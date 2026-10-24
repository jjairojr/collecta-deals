import Link from "next/link";
import { brl, pixelText } from "@/lib/format";
import { GAME_BLURB } from "@/lib/games";
import { setPath, type SetSummary } from "@/lib/sets";

export function CategoryIntro({ children }: { children: React.ReactNode }) {
  return <p className="mt-4 max-w-3xl text-[15px] leading-relaxed text-muted">{children}</p>;
}

export function GameIntro({ game, name, count }: { game: string; name: string; count: number }) {
  return (
    <CategoryIntro>
      {count > 0 ? `${count} cartas em estoque agora. ` : ""}
      {GAME_BLURB[game]} Pague por Pix ou cartão em até 12x, receba em todo o Brasil ou retire na loja em Goiânia.
    </CategoryIntro>
  );
}

export function SetIntro({ name, set }: { name: string; set: SetSummary }) {
  return (
    <CategoryIntro>
      {set.count} {set.count === 1 ? "carta avulsa" : "cartas avulsas"} da coleção {set.label} de {name} em
      estoque, a partir de {brl(set.minPrice)}. Cada carta é original, conferida uma a uma e enviada em sleeve e
      toploader para todo o Brasil.
    </CategoryIntro>
  );
}

export function SetLinks({
  game,
  name,
  sets,
  current,
}: {
  game: string;
  name: string;
  sets: SetSummary[];
  current?: string;
}) {
  const others = sets.filter((s) => s.code !== current);
  if (others.length === 0) {
    return null;
  }
  return (
    <section className="sticker mt-10 rounded-[14px] bg-surface p-6 sm:p-8 [--sh:6px]">
      <h2 className="font-display text-3xl font-bold leading-none text-white">
        {current ? `Outras coleções de ${name}` : `Coleções de ${name}`}
      </h2>
      <p className="mt-3 text-sm text-muted">Cartas avulsas por coleção, com preço a partir de:</p>
      <div className="mt-5 flex flex-wrap gap-3">
        {others.map((s) => (
          <Link
            key={s.code}
            href={setPath(game, s.code)}
            className="arcade-press sticker rounded-[10px] bg-royal px-4 py-2.5 font-pixel text-[9px] text-white [--sh:4px]"
          >
            {pixelText(s.label)} · {s.count} · {brl(s.minPrice)}
          </Link>
        ))}
      </div>
      <Link
        href={`/selado/${game}`}
        className="mt-6 inline-block font-pixel text-[9px] text-brand-soft underline-offset-4 hover:underline"
      >
        VER PRODUTO SELADO DE {pixelText(name.toUpperCase())} ›
      </Link>
    </section>
  );
}

export function GameFaq({ name }: { name: string }) {
  return (
    <section className="sticker mt-10 rounded-[14px] bg-surface p-6 sm:p-8 [--sh:6px]">
      <h2 className="font-display text-3xl font-bold leading-none text-white">Comprar cartas {name} avulsas</h2>
      <dl className="mt-5 grid gap-5 text-sm leading-relaxed text-[#c9c9d1] sm:grid-cols-2">
        <div>
          <dt className="font-bold text-white">As cartas {name} são originais?</dt>
          <dd className="mt-1">
            Sim. Só vendemos cartas oficiais, conferidas à mão quando entram no estoque e de novo antes do envio.
          </dd>
        </div>
        <div>
          <dt className="font-bold text-white">Como sei a condição da carta?</dt>
          <dd className="mt-1">
            Cada anúncio informa a condição (NM, LP, MP, HP). NM é Near Mint: sem marcas de uso visíveis.
          </dd>
        </div>
        <div>
          <dt className="font-bold text-white">Quanto custa o frete?</dt>
          <dd className="mt-1">
            O frete é calculado pelo CEP no checkout, com envio pelos Correios e código de rastreio. Em Goiânia dá para
            retirar na loja sem frete.
          </dd>
        </div>
        <div>
          <dt className="font-bold text-white">Quais as formas de pagamento?</dt>
          <dd className="mt-1">
            Pix ou cartão em até 12x pelo Mercado Pago, direto no site. Devolução em até 7 dias conforme a{" "}
            <Link href="/politica-de-devolucao" className="text-brand-soft underline-offset-4 hover:underline">
              política de trocas
            </Link>
            .
          </dd>
        </div>
      </dl>
    </section>
  );
}
