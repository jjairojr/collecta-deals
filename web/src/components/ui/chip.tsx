import type { ReactNode } from "react";

function Chip({ active, onClick, children }: { active: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={`rounded-[8px] border-2 border-outline px-3 py-1 text-xs font-bold transition-colors ${
        active ? "bg-sky-500/20 text-sky-200" : "bg-panel text-slate-400 hover:text-slate-200"
      }`}
    >
      {children}
    </button>
  );
}

export { Chip };
