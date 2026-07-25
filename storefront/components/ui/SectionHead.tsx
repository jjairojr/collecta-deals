import Link from "next/link";

// Section header: big Baloo H2 with an optional pixel eyebrow and an optional
// right-aligned "VER TODAS ›" link.
export default function SectionHead({
  title,
  eyebrow,
  action,
  size = "lg",
  heading = "h2",
}: {
  title: string;
  eyebrow?: string;
  action?: { label: string; href: string };
  size?: "lg" | "md";
  heading?: "h1" | "h2";
}) {
  const h2 = size === "lg" ? "text-4xl sm:text-[44px]" : "text-3xl sm:text-[38px]";
  const Tag = heading;
  return (
    <div className="mb-6 flex flex-wrap items-end justify-between gap-3">
      <div>
        {eyebrow && (
          <div className="mb-2 font-pixel text-[10px] text-brand">{eyebrow}</div>
        )}
        <Tag className={`font-display font-bold leading-none text-white ${h2}`}>
          {title}
        </Tag>
      </div>
      {action && (
        <Link
          href={action.href}
          className="inline-flex min-h-11 items-center font-pixel text-[10px] text-white"
        >
          <span className="border-b-[3px] border-brand pb-1">
            {action.label}
          </span>
        </Link>
      )}
    </div>
  );
}
