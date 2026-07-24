import { useState } from "react";
import { ChevronDown } from "lucide-react";
import { gameIsMultiLang, getGame, type CardSale, type LangSale } from "../api";
import { brl0 } from "../format";
import { langLabel, langTone } from "../lang";
import CardArt from "./CardArt";
import { Card } from "./ui/card";

function cleanName(n: string): string {
  return n.replace(/\s*\([^)]*\)\s*$/, "");
}

// LanguageSplit shows which printing actually moved. On Pokémon the Portuguese
// and English copies of a card are separate products at separate prices, so
// "4 sold" alone hides whether the demand was for the cheap PT print or the
// imported EN one. Only rendered for markets where the distinction exists.
function LanguageSplit({ languages }: { languages: LangSale[] }) {
  return (
    <div className="mt-1 flex flex-wrap gap-1">
      {languages.map((l) => (
        <span
          key={l.code}
          title={`${l.units} sold in ${langLabel(l.code)} · ${brl0(l.revenueBRL)} total · ${brl0(
            l.units > 0 ? l.revenueBRL / l.units : 0,
          )} avg`}
          className={`rounded px-1 py-px text-[10px] font-medium tabular-nums ring-1 ring-inset ${langTone(l.code)}`}
        >
          {langLabel(l.code)} ×{l.units}
        </span>
      ))}
    </div>
  );
}

export default function SoldCardTile({ card, set }: { card: CardSale; set: string }) {
  const [open, setOpen] = useState(false);
  const sellers = card.sellers ?? [];
  const languages = card.languages ?? [];
  const multiLang = gameIsMultiLang(getGame());
  const showLanguages = languages.length > 0 && multiLang;
  // Sellers are split per language, so one store can hold two rows; the header
  // counts shops, not rows.
  const storeCount = new Set(sellers.map((s) => s.storeId)).size;
  return (
    <Card className="flex flex-col overflow-hidden p-0">
      <div className="relative">
        <CardArt
          set={card.set ?? set}
          number={card.number}
          name={card.name}
          className="aspect-[350/489] w-full"
        />
        <div className="absolute left-2 top-2 rounded-lg bg-emerald-400 px-2 py-0.5 text-xs font-bold tabular-nums text-slate-950 shadow">
          {brl0(card.revenueBRL)}
        </div>
        <div className="absolute right-2 top-2 rounded-lg bg-slate-950/85 px-1.5 py-0.5 text-[11px] font-bold tabular-nums text-sky-200 ring-1 ring-inset ring-sky-500/30 backdrop-blur">
          ×{card.units}
        </div>
      </div>
      <div className="flex flex-1 flex-col p-2">
        <div className="truncate text-xs font-medium text-slate-100" title={card.name}>
          {cleanName(card.name)}
        </div>
        <div className="font-mono text-[10px] text-slate-500">
          {card.number}
          {card.set ? ` · ${card.set}` : ""}
        </div>
        {showLanguages && <LanguageSplit languages={languages} />}
        {sellers.length > 0 && (
          <div className="mt-auto pt-1.5">
            <button
              type="button"
              onClick={() => setOpen((o) => !o)}
              className="flex w-full items-center justify-between rounded-md bg-slate-800/60 px-2 py-1 text-[11px] text-slate-300 transition-colors hover:bg-slate-800"
            >
              <span>
                {storeCount} store{storeCount === 1 ? "" : "s"}
              </span>
              <ChevronDown className={`h-3 w-3 transition-transform ${open ? "rotate-180" : ""}`} />
            </button>
            {open && (
              <ul className="mt-1 space-y-0.5">
                {sellers.map((s) => (
                  <li
                    key={`${s.storeId}-${s.language ?? ""}`}
                    className="flex items-center justify-between gap-2 rounded px-1.5 py-0.5 text-[11px] hover:bg-slate-800/40"
                  >
                    <span className="truncate text-slate-400" title={s.storeName}>
                      {s.storeName}
                    </span>
                    <span className="flex shrink-0 items-center gap-1.5 tabular-nums">
                      {multiLang && s.language && (
                        <span
                          className={`rounded px-1 text-[9px] font-medium ring-1 ring-inset ${langTone(s.language)}`}
                          title={`sold in ${langLabel(s.language)}`}
                        >
                          {langLabel(s.language)}
                        </span>
                      )}
                      {s.priceBRL > 0 && (
                        <span className="text-emerald-300" title="price this store sold at">
                          {brl0(s.priceBRL)}
                        </span>
                      )}
                      <span className="font-medium text-sky-300">×{s.units}</span>
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        )}
      </div>
    </Card>
  );
}
