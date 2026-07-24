"use client";

// Multi-select filter chip: 3px ink border, pink when selected else dark with
// muted text. Also used for the sort toggle.
export default function FilterChip({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      aria-pressed={active}
      onClick={onClick}
      className={`rounded-[8px] border-[3px] border-outline px-2.5 py-1.5 text-xs font-bold transition-colors ${
        active ? "bg-brand text-white" : "bg-outline text-[#c9c9d1] hover:text-white"
      }`}
    >
      {children}
    </button>
  );
}
