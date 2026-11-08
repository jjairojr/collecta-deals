import Link from "next/link";
import { brl, pixelText } from "@/lib/format";
import {
  collectionPath,
  contentsOf,
  formatsIn,
  MIN_COLLECTION_ITEMS,
  releaseLabel,
  type SealedCollection,
} from "@/lib/sealed-guide";
import type { Sealed } from "@/lib/types";

const panel = "sticker mt-10 rounded-[14px] bg-surface p-6 sm:p-8 [--sh:6px]";

export function SealedCollectionAbout({
  collection,
  item,
  siblings,
}: {
  collection: SealedCollection;
  item: Sealed;
  siblings: number;
}) {
  const contents = contentsOf(item);
  return (
    <section className={panel}>
      <h2 className="font-display text-3xl font-bold leading-none text-white">Sobre {collection.short}</h2>
      {contents && (
        <div className="mt-5">
          <h3 className="font-pixel text-[10px] text-brand-soft">O QUE VEM</h3>
          <p className="mt-2 text-[15px] leading-relaxed text-white">{contents}</p>
        </div>
      )}
      <p className="mt-5 max-w-3xl text-[15px] leading-relaxed text-muted">{collection.summary}</p>
      {(collection.released || collection.nameEN) && (
        <dl className="mt-4 flex flex-wrap gap-x-8 gap-y-2 text-sm text-muted">
          {collection.released && (
            <div>
              <dt className="inline font-bold text-white">Lançamento: </dt>
              <dd className="inline">{releaseLabel(collection.released)}</dd>
            </div>
          )}
          {collection.nameEN && collection.nameEN !== collection.short && (
            <div>
              <dt className="inline font-bold text-white">Nome em inglês: </dt>
              <dd className="inline">{collection.nameEN}</dd>
            </div>
          )}
        </dl>
      )}
      {siblings >= MIN_COLLECTION_ITEMS && (
        <Link
          href={collectionPath(collection)}
          className="mt-6 inline-block font-pixel text-[9px] text-brand-soft underline-offset-4 hover:underline"
        >
          VER TODOS OS PRODUTOS DE {pixelText(collection.short)} ›
        </Link>
      )}
    </section>
  );
}

export function SealedCollectionLinks({
  title,
  collections,
}: {
  title: string;
  collections: { collection: SealedCollection; items: Sealed[] }[];
}) {
  if (collections.length === 0) {
    return null;
  }
  return (
    <section className={panel}>
      <h2 className="font-display text-3xl font-bold leading-none text-white">{title}</h2>
      <p className="mt-3 text-sm text-muted">Todos os produtos de cada coleção, com preço e o que vem em cada caixa:</p>
      <div className="mt-5 flex flex-wrap gap-3">
        {collections.map(({ collection, items }) => (
          <Link
            key={collection.slug}
            href={collectionPath(collection)}
            className="arcade-press sticker rounded-[10px] bg-royal px-4 py-2.5 font-pixel text-[9px] text-white [--sh:4px]"
          >
            {pixelText(collection.short)} · {items.length} · {brl(Math.min(...items.map((s) => s.price)))}
          </Link>
        ))}
      </div>
    </section>
  );
}

export function SealedFormatGuide({ title, list }: { title: string; list: Sealed[] }) {
  const formats = formatsIn(list);
  if (formats.length === 0) {
    return null;
  }
  return (
    <section className={panel}>
      <h2 className="font-display text-3xl font-bold leading-none text-white">{title}</h2>
      <dl className="mt-5 grid gap-5 text-sm leading-relaxed text-[#c9c9d1] sm:grid-cols-2">
        {formats.map((f) => (
          <div key={f.label}>
            <dt className="font-bold text-white">{f.label}</dt>
            <dd className="mt-1">{f.text}</dd>
          </div>
        ))}
      </dl>
    </section>
  );
}
