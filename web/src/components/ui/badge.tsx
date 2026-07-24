import { type HTMLAttributes } from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-[6px] border-2 border-outline px-1.5 py-0.5 text-[11px] font-bold",
  {
    variants: {
      variant: {
        default: "bg-surface text-slate-200",
        sky: "bg-sky-500/20 text-sky-300",
        emerald: "bg-emerald-500/20 text-emerald-300",
        amber: "bg-amber-500/20 text-amber-300",
        violet: "bg-violet-500/20 text-violet-300",
        fuchsia: "bg-fuchsia-500/20 text-fuchsia-300",
        rose: "bg-rose-500/20 text-rose-300",
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
