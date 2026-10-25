"use client";

import { pixelText } from "@/lib/format";

// Arcade text input: pixel label on top, white sticker body (same shell as the
// quantity stepper), pink error line underneath.
export default function Field({
  label,
  name,
  value,
  onChange,
  onBlur,
  onEnter,
  error,
  placeholder,
  inputMode,
  autoComplete,
  maxLength,
  className = "",
}: {
  label: string;
  name: string;
  value: string;
  onChange: (v: string) => void;
  onBlur?: () => void;
  onEnter?: () => void;
  error?: string;
  placeholder?: string;
  inputMode?: "text" | "numeric" | "tel" | "email";
  autoComplete?: string;
  maxLength?: number;
  className?: string;
}) {
  return (
    <label className={`block min-w-0 ${className}`}>
      <span className="mb-1.5 block font-pixel text-[8px] text-brand-soft">
        {pixelText(label)}
      </span>
      <span
        className={`sticker flex overflow-hidden rounded-[10px] bg-white ${
          error ? "border-brand" : ""
        }`}
        style={{ ["--sh" as string]: "0px", borderWidth: 3 }}
      >
        <input
          name={name}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onBlur={onBlur}
          onKeyDown={
            onEnter
              ? (e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    onEnter();
                  }
                }
              : undefined
          }
          placeholder={placeholder}
          inputMode={inputMode}
          autoComplete={autoComplete}
          maxLength={maxLength}
          aria-label={label}
          aria-invalid={error ? true : undefined}
          className="min-w-0 flex-1 bg-transparent px-3 py-2.5 text-base text-outline outline-none placeholder:text-faint sm:text-sm"
        />
      </span>
      {error && (
        <span className="mt-1 block font-pixel text-[7px] text-brand">
          {pixelText(error)}
        </span>
      )}
    </label>
  );
}
