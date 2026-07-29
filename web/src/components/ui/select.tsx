import { forwardRef, type SelectHTMLAttributes } from "react";
import { ChevronDown } from "lucide-react";
import { cn } from "../../lib/utils";

const Select = forwardRef<HTMLSelectElement, SelectHTMLAttributes<HTMLSelectElement>>(
  ({ className, children, ...props }, ref) => (
    <div className="relative inline-flex w-full items-center">
      <select
        ref={ref}
        className={cn(
          "h-9 w-full appearance-none rounded-[8px] border-2 border-outline bg-panel pl-3 pr-9 text-sm font-medium text-fg transition-colors disabled:cursor-not-allowed disabled:opacity-50",
          className,
        )}
        {...props}
      >
        {children}
      </select>
      <ChevronDown className="pointer-events-none absolute right-3 h-4 w-4 text-slate-500" />
    </div>
  ),
);
Select.displayName = "Select";

export { Select };
