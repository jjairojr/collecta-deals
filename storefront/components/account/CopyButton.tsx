"use client";

import { useState } from "react";

export default function CopyButton({ text, label = "COPIAR" }: { text: string; label?: string }) {
  const [done, setDone] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setDone(true);
      setTimeout(() => setDone(false), 1500);
    } catch {
      setDone(false);
    }
  };

  return (
    <button
      type="button"
      onClick={() => void copy()}
      className="arcade-press sticker rounded-[10px] bg-brand-soft px-4 py-2.5 font-pixel text-[9px] text-outline [--sh:4px]"
    >
      {done ? "COPIADO!" : label}
    </button>
  );
}
