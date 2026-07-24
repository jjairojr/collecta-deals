import { HIGH_SCORES } from "@/lib/mock";

// Leaderboard panel — dark surface, pink border. Trade counts/scores are mock.
export default function HighScores() {
  return (
    <section
      className="sticker sticker-5 overflow-hidden rounded-[18px] border-brand bg-outline"
      style={{ ["--sh" as string]: "8px" }}
    >
      <div className="px-6 pt-7 sm:px-8">
        <h2 className="font-pixel text-[18px] text-brand-soft">HIGH SCORES</h2>
        <p className="mt-2 text-[13px] text-faint">
          Colecionadores com mais trocas neste mês
        </p>
      </div>
      <div className="mt-5 pb-2">
        {HIGH_SCORES.map((row, i) => (
          <div
            key={row.player}
            className={`grid grid-cols-[46px_1fr_auto] items-center gap-3 px-6 py-3.5 font-pixel text-[11px] sm:grid-cols-[50px_1fr_130px_110px] sm:gap-4 sm:px-8 ${
              i % 2 === 0 ? "bg-surface" : "bg-ink"
            }`}
          >
            <span className="text-brand">{row.position}</span>
            <span className="truncate text-white">{row.player}</span>
            <span className="hidden text-brand-soft sm:block">
              {row.trades} TROCAS
            </span>
            <span className="text-right text-white">
              {row.score.toLocaleString("pt-BR")}
            </span>
          </div>
        ))}
      </div>
    </section>
  );
}
