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
        "inline-flex rounded-[8px] border-[3px] border-outline bg-outline p-0.5",
        className,
      )}
    >
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          onClick={() => onChange(o.value)}
          className={cn(
            "rounded-[6px] px-3 py-1.5 text-xs font-bold transition-colors",
            value === o.value
              ? "bg-brand text-white"
              : "text-slate-400 hover:text-white",
          )}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}
