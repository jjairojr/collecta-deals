import Link from "next/link";
import { guidePath, type Guide } from "@/lib/guides";

export default function GuideLinks({ guides, title = "Guias" }: { guides: Guide[]; title?: string }) {
  if (guides.length === 0) {
    return null;
  }
  return (
    <section className="sticker mt-10 rounded-[14px] bg-surface p-6 sm:p-8 [--sh:6px]">
      <h2 className="font-display text-3xl font-bold leading-none text-white">{title}</h2>
      <ul className="mt-5 grid gap-x-8 gap-y-3 text-[15px] sm:grid-cols-2">
        {guides.map((g) => (
          <li key={g.slug}>
            <Link href={guidePath(g)} className="text-brand-soft underline-offset-4 hover:underline">
              {g.title}
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}
