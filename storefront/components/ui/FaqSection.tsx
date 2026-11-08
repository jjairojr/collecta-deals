import JsonLd from "@/components/seo/JsonLd";
import { faqJsonLd, type FaqItem } from "@/lib/seo";

export default function FaqSection({
  title = "Perguntas frequentes",
  items,
  schema = true,
}: {
  title?: string;
  items: FaqItem[];
  schema?: boolean;
}) {
  if (items.length === 0) {
    return null;
  }
  return (
    <section className="sticker mt-10 rounded-[14px] bg-surface p-6 sm:p-8 [--sh:6px]">
      {schema && <JsonLd data={faqJsonLd(items)} />}
      <h2 className="font-display text-3xl font-bold leading-none text-white">{title}</h2>
      <dl className="mt-5 grid gap-5 text-sm leading-relaxed text-[#c9c9d1] sm:grid-cols-2">
        {items.map((it) => (
          <div key={it.q}>
            <dt className="font-bold text-white">{it.q}</dt>
            <dd className="mt-1">{it.a}</dd>
          </div>
        ))}
      </dl>
    </section>
  );
}
