// Liga idioma codes, read off the language <select> on a Liga card page. Liga
// adds codes over time (Pokémon snapshots already carry ones absent from that
// list), so anything unmapped falls back to the raw code rather than vanishing.
const LIGA_LANGS: Record<string, string> = {
  "1": "DE",
  "2": "EN",
  "3": "ES",
  "4": "FR",
  "5": "IT",
  "6": "JP",
  "7": "KO",
  "8": "PT",
  "9": "RU",
  "10": "ZH",
  "11": "PT/EN",
};

export function langLabel(code: string): string {
  return LIGA_LANGS[code] ?? (code ? `#${code}` : "?");
}

// langTone colors the two languages a Brazilian buyer actually chooses between —
// the Portuguese print and the imported English one. Everything else stays
// neutral so a rare Japanese or Chinese copy reads as an aside, not a signal.
export function langTone(code: string): string {
  if (code === "8") {
    return "bg-emerald-500/10 text-emerald-300 ring-emerald-500/25";
  }
  if (code === "2") {
    return "bg-sky-500/10 text-sky-300 ring-sky-500/25";
  }
  return "bg-slate-800/70 text-slate-400 ring-slate-700";
}
