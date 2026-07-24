import type { ReactNode } from "react";
import type { Icon } from "../brand";
import { cn } from "../lib/utils";

export default function PageHeader({
  title,
  description,
  icon: IconCmp,
  actions,
  className,
}: {
  title: string;
  description?: string;
  icon?: Icon;
  actions?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex flex-wrap items-start justify-between gap-4", className)}>
      <div className="flex items-center gap-3">
        {IconCmp && (
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-[10px] border-[3px] border-outline bg-brand text-white shadow-[3px_3px_0_#0b0b0c]">
            <IconCmp className="h-5 w-5" />
          </div>
        )}
        <div>
          <h1 className="font-display text-2xl font-extrabold tracking-tight text-white">{title}</h1>
          {description && <p className="mt-0.5 text-sm text-slate-400">{description}</p>}
        </div>
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </div>
  );
}
