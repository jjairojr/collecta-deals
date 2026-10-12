"use client";

import { useState } from "react";
import { ChevronDown } from "lucide-react";
import { UFS } from "@/lib/checkout";
import { pixelText } from "@/lib/format";

// State picker in the arcade chip style (no native select): white sticker
// trigger matching the text fields, sticker panel with the 27 UFs.
export default function UFSelect({
  value,
  onChange,
  error,
}: {
  value: string;
  onChange: (uf: string) => void;
  error?: string;
}) {
  const [open, setOpen] = useState(false);
  const pick = (uf: string) => {
    onChange(uf);
    setOpen(false);
  };
  return (
    <div className="relative min-w-0">
      <span className="mb-1.5 block font-pixel text-[8px] text-brand-soft">UF</span>
      <button
        type="button"
        aria-expanded={open}
        aria-label="Estado"
        onClick={() => setOpen((o) => !o)}
        className={`sticker flex w-full items-center justify-between gap-2 rounded-[10px] bg-white px-3 py-2.5 text-left text-base font-bold text-outline sm:text-sm ${
          error ? "border-brand" : ""
        }`}
        style={{ ["--sh" as string]: "0px", borderWidth: 3 }}
      >
        <span className={value ? "" : "text-faint"}>{value || "UF"}</span>
        <ChevronDown
          className={`h-3.5 w-3.5 shrink-0 transition-transform ${open ? "rotate-180" : ""}`}
        />
      </button>
      {error && (
        <span className="mt-1 block font-pixel text-[7px] text-brand">
          {pixelText(error)}
        </span>
      )}
      {open && (
        <>
          <button
            type="button"
            aria-label="Fechar estados"
            onClick={() => setOpen(false)}
            className="fixed inset-0 z-10 cursor-default"
          />
          <div className="sticker absolute inset-x-0 top-full z-20 mt-2 grid max-h-60 grid-cols-3 gap-1 overflow-y-auto rounded-[10px] bg-surface p-1.5 [--sh:4px]">
            {UFS.map((uf) => (
              <button
                key={uf}
                type="button"
                onClick={() => pick(uf)}
                className={`rounded-[6px] px-2 py-2 text-center text-xs font-bold transition-colors ${
                  uf === value
                    ? "bg-brand text-white"
                    : "text-[#c9c9d1] hover:bg-outline hover:text-white"
                }`}
              >
                {uf}
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
