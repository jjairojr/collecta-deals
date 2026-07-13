import { forwardRef, type InputHTMLAttributes } from "react";
import { cn } from "../../lib/utils";

export interface CheckboxProps extends InputHTMLAttributes<HTMLInputElement> {
  accent?: "sky" | "emerald";
}

const Checkbox = forwardRef<HTMLInputElement, CheckboxProps>(
  ({ className, accent = "sky", ...props }, ref) => (
    <input
      ref={ref}
      type="checkbox"
      className={cn(
        "h-4 w-4 cursor-pointer rounded border-slate-600",
        accent === "emerald" ? "accent-emerald-500" : "accent-sky-500",
        className,
      )}
      {...props}
    />
  ),
);
Checkbox.displayName = "Checkbox";

export { Checkbox };
