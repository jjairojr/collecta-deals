import { useEffect, useMemo, useState } from "react";
import { Check } from "lucide-react";
import { cardImageURL, getCards, type TrackCard } from "../api";
import { brl } from "../format";
import { useSets } from "../useSets";
import { useSelection } from "../selection";
import SetSelect from "./SetSelect";
import { Card } from "./ui/card";
import { Input } from "./ui/input";

export default function BrowsePage() {
  const sets = useSets();
  const [set, setSet] = useState("");
  const [cards, setCards] = useState<TrackCard[]>([]);
  const [q, setQ] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { has, toggle, backfill } = useSelection();

  useEffect(() => {
    if (sets.length > 0 && !sets.includes(set)) {
      setSet(sets[0]);
    }
  }, [sets, set]);

  useEffect(() => {
    backfill(cards.map((c) => ({ set, number: c.number, name: c.name, priceBRL: c.lowBRL })));
  }, [cards, set, backfill]);

  useEffect(() => {
    if (!set) {
      return;
    }
    let current = true;
    setLoading(true);
    setError(null);
    getCards(set)
      .then((r) => {
        if (current) {
          setCards(r.cards);
        }
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "failed to load cards"))
      .finally(() => current && setLoading(false));
    return () => {
      current = false;
    };
  }, [set]);

  const filtered = useMemo(() => {
    const t = q.trim().toLowerCase();
    if (!t) {
      return cards;
    }
    return cards.filter(
      (c) => c.name.toLowerCase().includes(t) || c.number.toLowerCase().includes(t),
    );
  }, [cards, q]);

  return (
    <div className="space-y-4 pb-24">
      <div>
        <h1 className="text-lg font-semibold text-slate-100">Browse cards</h1>
        <p className="mt-1 max-w-2xl text-sm text-slate-400">
          Tap cards you want to buy, then export a flyer image and copy the message from the bar
          below to post in your WhatsApp group.
        </p>
      </div>

      <Card className="flex flex-wrap items-end gap-4 p-4">
        <SetSelect sets={sets} value={set} onChange={setSet} />
        <label className="flex min-w-[200px] flex-1 flex-col gap-1 text-xs text-slate-400">
          Search
          <Input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="name or number — e.g. Luffy, OP16-056"
          />
        </label>
        <div className="text-xs text-slate-500">
          {filtered.length} card{filtered.length === 1 ? "" : "s"}
          {loading && <span className="ml-2">loading…</span>}
        </div>
      </Card>

      {error && (
        <Card className="px-4 py-10 text-center text-slate-400">Could not load: {error}</Card>
      )}

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6">
        {filtered.map((c) => {
          const picked = has(set, c.number);
          return (
            <button
              key={c.number}
              onClick={() => toggle({ set, number: c.number, name: c.name, priceBRL: c.lowBRL })}
              title={c.name}
              className={`group relative overflow-hidden rounded-lg border bg-slate-900 text-left transition ${
                picked
                  ? "border-emerald-400 ring-2 ring-emerald-400/40"
                  : "border-slate-800 hover:border-slate-600"
              }`}
            >
              <img
                src={cardImageURL(set, c.number)}
                alt={c.name}
                loading="lazy"
                className="aspect-[350/489] w-full bg-slate-800 object-cover"
              />
              <span
                className={`absolute right-1.5 top-1.5 flex h-5 w-5 items-center justify-center rounded-full border ${
                  picked
                    ? "border-emerald-400 bg-emerald-400 text-slate-950"
                    : "border-slate-500 bg-slate-900/70 text-transparent group-hover:text-slate-500"
                }`}
              >
                <Check className="h-3 w-3" strokeWidth={3} />
              </span>
              <span className="block truncate px-2 py-1 font-mono text-[11px] text-slate-300">
                {c.number} · {brl(c.lowBRL)}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
