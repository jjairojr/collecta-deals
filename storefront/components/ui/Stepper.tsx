"use client";

// Arcade quantity stepper: white sticker body, light-pink −/+ pads, pixel value.
// Floors at `min` (default 1) and caps at `max`.
export default function Stepper({
  value,
  onChange,
  min = 1,
  max = 99,
  size = "md",
}: {
  value: number;
  onChange: (n: number) => void;
  min?: number;
  max?: number;
  size?: "sm" | "md";
}) {
  const pad = size === "sm" ? "px-2.5 py-1.5" : "px-3 py-2";
  const val = size === "sm" ? "min-w-6 text-[12px]" : "min-w-7 text-[14px]";
  return (
    <div
      className="sticker flex items-stretch overflow-hidden rounded-[10px] bg-white"
      style={{ ["--sh" as string]: "0px" }}
    >
      <button
        type="button"
        aria-label="Diminuir quantidade"
        disabled={value <= min}
        onClick={() => onChange(Math.max(min, value - 1))}
        className="border-r-4 border-outline bg-brand-soft font-pixel text-[14px] text-outline disabled:opacity-40 px-3"
      >
        −
      </button>
      <span
        className={`grid place-items-center font-pixel text-outline ${val} ${pad}`}
      >
        {value}
      </span>
      <button
        type="button"
        aria-label="Aumentar quantidade"
        disabled={value >= max}
        onClick={() => onChange(Math.min(max, value + 1))}
        className="border-l-4 border-outline bg-brand-soft font-pixel text-[14px] text-outline disabled:opacity-40 px-3"
      >
        +
      </button>
    </div>
  );
}
