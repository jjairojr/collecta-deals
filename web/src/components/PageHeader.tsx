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
          <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-slate-800 bg-gradient-to-br from-accent-500/20 to-accent-500/5 text-accent-300 ring-1 ring-inset ring-accent-500/20">
            <IconCmp className="h-5 w-5" />
          </div>
        )}
        <div>
          <h1 className="text-xl font-semibold tracking-tight text-slate-100">{title}</h1>
          {description && <p className="mt-0.5 text-sm text-slate-400">{description}</p>}
        </div>
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </div>
  );
}
