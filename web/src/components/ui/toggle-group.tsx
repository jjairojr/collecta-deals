import { type ReactNode } from "react";
import { cn } from "../../lib/utils";

export interface ToggleOption {
  value: string;
  label: ReactNode;
}

export function ToggleGroup({
  value,
  options,
  onChange,
  className,
}: {
  value: string;
  options: ToggleOption[];
  onChange: (value: string) => void;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "pill pill-sm inline-flex bg-raised p-0.5",
        className,
      )}
    >
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          onClick={() => onChange(o.value)}
          className={cn(
            "rounded-full border-2 px-3 py-1 text-xs font-bold transition-colors",
            value === o.value
              ? "border-outline bg-brand text-white"
              : "border-transparent text-slate-400 hover:text-fg",
          )}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}
