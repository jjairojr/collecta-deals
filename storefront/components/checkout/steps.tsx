"use client";

import { Check } from "lucide-react";
import { pixelText } from "@/lib/format";

export type StepNumber = 1 | 2 | 3 | 4;

export const STEPS: { n: StepNumber; label: string }[] = [
  { n: 1, label: "Dados" },
  { n: 2, label: "Entrega" },
  { n: 3, label: "Pagamento" },
  { n: 4, label: "Revisão" },
];

// The four steps across the top. A finished step can be tapped to go back to it;
// the ones ahead stay locked until the current step validates.
export function StepIndicator({
  current,
  reached,
  onGo,
}: {
  current: StepNumber;
  reached: StepNumber;
  onGo: (n: StepNumber) => void;
}) {
  return (
    <ol className="grid grid-cols-4 gap-2" aria-label="Etapas do checkout">
      {STEPS.map((s) => {
        const done = s.n < current;
        const active = s.n === current;
        const canGo = s.n <= reached && !active;
        return (
          <li key={s.n}>
            <button
              type="button"
              onClick={() => canGo && onGo(s.n)}
              disabled={!canGo}
              aria-current={active ? "step" : undefined}
              className={`sticker flex min-h-[52px] w-full flex-col items-center justify-center gap-1 rounded-[10px] px-1 py-2 text-center [--sh:3px] ${
                active
                  ? "bg-brand text-white"
                  : done
                    ? "bg-outline text-brand-soft hover:text-white"
                    : "bg-outline text-faint"
              } disabled:cursor-default`}
            >
              <span className="flex items-center gap-1 font-pixel text-[10px]">
                {done ? <Check size={12} strokeWidth={3} aria-hidden /> : s.n}
              </span>
              <span className="text-[11px] leading-none sm:text-xs">{s.label}</span>
            </button>
          </li>
        );
      })}
    </ol>
  );
}

export function StepCard({
  step,
  title,
  headingRef,
  children,
}: {
  step: StepNumber;
  title: string;
  headingRef?: React.Ref<HTMLHeadingElement>;
  children: React.ReactNode;
}) {
  return (
    <section className="sticker rounded-[14px] bg-surface p-5 sm:p-6 [--sh:5px]">
      <div className="mb-5 flex items-baseline gap-3">
        <span className="font-pixel text-[9px] text-brand">{`0${step}`}</span>
        <h2 ref={headingRef} tabIndex={-1} className="font-display text-2xl font-bold leading-none text-white outline-none">
          {title}
        </h2>
      </div>
      {children}
    </section>
  );
}

// A large tap target for one choice in a radio group (shipping, payment). Price
// sits on the right, an optional struck-through "was" price under it.
export function ChoiceCard({
  title,
  lines,
  price,
  was,
  selected,
  onSelect,
}: {
  title: string;
  lines: React.ReactNode[];
  price?: string;
  was?: string;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      onClick={onSelect}
      className={`sticker flex min-h-[64px] w-full items-center gap-3 rounded-[12px] px-4 py-3 text-left transition-colors [--sh:3px] ${
        selected ? "bg-brand text-white" : "bg-outline text-[#c9c9d1] hover:text-white"
      }`}
    >
      <span
        aria-hidden
        className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-full border-[3px] ${
          selected ? "border-white bg-white text-brand" : "border-[#6b6b74]"
        }`}
      >
        {selected && <span className="h-2.5 w-2.5 rounded-full bg-brand" />}
      </span>
      <span className="min-w-0 flex-1">
        <span className="block font-pixel text-[9px] leading-relaxed">{pixelText(title)}</span>
        {lines.map((l, i) => (
          <span key={i} className={`block text-[13px] leading-snug ${selected ? "text-white/90" : "text-muted"}`}>
            {l}
          </span>
        ))}
      </span>
      {price && (
        <span className="flex shrink-0 flex-col items-end">
          <span className="font-pixel text-[12px]">{price}</span>
          {was && <span className={`text-xs line-through ${selected ? "text-white/70" : "text-faint"}`}>{was}</span>}
        </span>
      )}
    </button>
  );
}

// Review block in step 4: what was chosen, with a link back to change it.
export function ReviewBlock({
  title,
  onEdit,
  children,
}: {
  title: string;
  onEdit: () => void;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-[12px] border-[3px] border-outline p-4">
      <div className="mb-2 flex items-center justify-between gap-3">
        <h3 className="font-pixel text-[9px] text-brand-soft">{pixelText(title)}</h3>
        <button
          type="button"
          onClick={onEdit}
          className="min-h-[44px] px-2 text-sm text-brand-soft underline-offset-4 hover:underline"
        >
          Alterar
        </button>
      </div>
      <div className="space-y-1 text-sm text-white">{children}</div>
    </div>
  );
}
