import { brl } from "@/lib/format";
import type { MarketRow } from "@/lib/market";

function dayLabel(date: string): string {
  const [y, m, d] = date.slice(0, 10).split("-");
  return `${d}/${m}/${y}`;
}

export default function MarketPrice({ rows, cardLabel }: { rows: MarketRow[]; cardLabel: string }) {
  if (rows.length === 0) {
    return null;
  }
  return (
    <section
      className="sticker mt-5 rounded-[14px] bg-surface p-5"
      style={{ ["--sh" as string]: "6px" }}
      aria-label={`Preço de mercado de ${cardLabel}`}
    >
      <h2 className="font-pixel text-[10px] text-brand-soft">PRECO DE MERCADO NO BRASIL</h2>
      <p className="mt-2 text-[13px] text-muted">
        Quanto vale {cardLabel} hoje: menor preço anunciado em condição NM e a mediana dos últimos 30 dias.
      </p>
      <table className="mt-4 w-full text-left text-[14px]">
        <thead>
          <tr className="font-pixel text-[8px] text-faint">
            <th className="pb-2 font-normal">IDIOMA</th>
            <th className="pb-2 font-normal">MENOR HOJE</th>
            <th className="pb-2 font-normal">MEDIANA 30 DIAS</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.language} className="border-t-2 border-outline text-white">
              <td className="py-2">{r.language}</td>
              <td className="py-2 font-bold">{brl(Math.round(r.lowBRL * 100))}</td>
              <td className="py-2">{brl(Math.round(r.medianBRL * 100))}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <p className="mt-3 text-[11px] text-faint">Atualizado em {dayLabel(rows[0].date)}.</p>
    </section>
  );
}
