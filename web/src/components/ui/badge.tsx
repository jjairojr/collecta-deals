import { type HTMLAttributes } from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded px-1.5 py-0.5 text-xs font-semibold ring-1 ring-inset",
  {
    variants: {
      variant: {
        default: "bg-slate-500/15 text-slate-300 ring-slate-500/30",
        sky: "bg-sky-500/15 text-sky-300 ring-sky-500/30",
        emerald: "bg-emerald-500/15 text-emerald-300 ring-emerald-500/30",
        amber: "bg-amber-500/15 text-amber-300 ring-amber-500/30",
        violet: "bg-violet-500/15 text-violet-300 ring-violet-500/30",
        fuchsia: "bg-fuchsia-500/15 text-fuchsia-300 ring-fuchsia-500/30",
        rose: "bg-rose-500/15 text-rose-300 ring-rose-500/30",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  },
);

export interface BadgeProps
  extends HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeVariants({ variant }), className)} {...props} />;
}

export { Badge, badgeVariants };
