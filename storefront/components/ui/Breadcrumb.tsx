import Link from "next/link";

// Pixel breadcrumb. Segments must be accent-free (Press Start 2P has no accents)
// — callers pass already-normalized uppercase labels. A segment with an href
// becomes a link; the last segment is the current page and stays plain.
export interface Crumb {
  label: string;
  href?: string;
}

export default function Breadcrumb({ items }: { items: Crumb[] }) {
  return (
    <nav aria-label="Trilha" className="font-pixel text-[9px] leading-relaxed text-faint">
      {items.map((it, i) => {
        const last = i === items.length - 1;
        return (
          <span key={it.label}>
            {i > 0 && <span className="mx-1.5 text-faint">{">"}</span>}
            {it.href && !last ? (
              <Link href={it.href} className="text-faint transition-colors hover:text-brand">
                {it.label}
              </Link>
            ) : (
              <span className={last ? "text-brand-soft" : ""}>{it.label}</span>
            )}
          </span>
        );
      })}
    </nav>
  );
}
