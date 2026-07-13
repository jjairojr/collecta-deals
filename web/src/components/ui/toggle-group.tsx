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
        "inline-flex rounded-lg border border-slate-700 bg-slate-900 p-0.5",
        className,
      )}
    >
      {options.map((o) => (
        <button
          key={o.value}
          type="button"
          onClick={() => onChange(o.value)}
          className={cn(
            "rounded-md px-3 py-1.5 text-xs font-medium transition-colors",
            value === o.value
              ? "bg-sky-500/20 text-sky-200"
              : "text-slate-400 hover:text-slate-200",
          )}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}
