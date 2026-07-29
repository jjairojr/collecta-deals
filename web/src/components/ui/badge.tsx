import { type HTMLAttributes } from "react";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "../../lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-full border-2 border-outline px-2 py-0.5 text-[11px] font-bold",
  {
    variants: {
      variant: {
        default: "bg-raised text-fg",
        sky: "bg-brand-soft text-on-soft",
        emerald: "bg-emerald-500/15 text-emerald-300",
        amber: "bg-amber-500/15 text-amber-300",
        violet: "bg-brand-soft text-on-soft",
        fuchsia: "bg-brand-soft text-on-soft",
        rose: "bg-rose-500/15 text-rose-300",
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
