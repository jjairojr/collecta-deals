import { useCallback, useEffect, useMemo, useState } from "react";
import { AlertTriangle, Check, Download, FileUp, Info, Loader2, X } from "lucide-react";
import { getGame, getPortfolio, type TradeView } from "../api";
import { brl } from "../format";
import EmptyState from "./EmptyState";

const COL = {
  edicaoId: 1,
  sigla: 2,
  cartaId: 3,
  numero: 4,
  nomePT: 8,
  nomeEN: 9,
  idioma: 10,
  qualidade: 11,
  qtdSomar: 13,
  preco: 14,
} as const;

const INGLES = /\(\s*ingl?[eê]s\s*\)/i;
const NUMERO_NO_NOME = /\(#?(\d+)\s*\/\s*\d+\)/;
const QUALIDADES = new Set(["M", "NM", "SP", "MP", "HP", "D"]);

type Linha = string[];

interface Catalogo {
  header: Linha[];
  porEdicao: Map<string, Linha[]>;
  arquivos: { nome: string; sigla: string; cards: number }[];
}

interface Resultado {
  trade: TradeView;
  candidatos: Linha[];
  escolhido: number;
  idioma: "PT" | "EN";
  qualidade: string;
}

function parseCSV(texto: string): Linha[] {
  const linhas: Linha[] = [];
  let campo = "";
  let linha: string[] = [];
  let aspas = false;
  const src = texto.replace(/^﻿/, "");
  for (let i = 0; i < src.length; i++) {
    const c = src[i];
    if (aspas) {
      if (c === '"') {
        if (src[i + 1] === '"') {
          campo += '"';
          i++;
        } else aspas = false;
      } else campo += c;
      continue;
    }
    if (c === '"') aspas = true;
    else if (c === ",") {
      linha.push(campo);
      campo = "";
    } else if (c === "\n") {
      linha.push(campo);
      linhas.push(linha);
      linha = [];
      campo = "";
    } else if (c !== "\r") campo += c;
  }
  if (campo || linha.length) {
    linha.push(campo);
    linhas.push(linha);
  }
  return linhas.filter((l) => l.length > 1);
}

function semAcento(s: string): string {
  return s
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]/g, "");
}

function chaveNome(nome: string): string {
  return semAcento(nome.replace(/\(.*?\)/g, ""));
}

function casar(trade: TradeView, catalogo: Catalogo): Linha[] {
  const todas = [...catalogo.porEdicao.values()].flat();
  const sigla = (trade.set || "").trim();
  const daEdicao = todas.filter((l) => {
    const s = l[COL.sigla];
    return sigla === s || sigla.includes(`(${s})`);
  });
  const escopo = daEdicao.length ? daEdicao : todas;

  const m = NUMERO_NO_NOME.exec(trade.name || "");
  const numero = (trade.number || "").trim() || (m ? m[1] : "");
  if (numero) {
    const porNumero = escopo.filter((l) => {
      const n = l[COL.numero].trim();
      return /^\d+$/.test(n) && parseInt(n, 10) === parseInt(numero, 10);
    });
    if (porNumero.length) return porNumero;
  }
  const alvo = chaveNome(trade.name || "");
  return escopo.filter((l) => semAcento(l[COL.nomePT]) === alvo || semAcento(l[COL.nomeEN]) === alvo);
}

export default function LigaExportPage() {
  const [trades, setTrades] = useState<TradeView[] | null>(null);
  const [catalogo, setCatalogo] = useState<Catalogo | null>(null);
  const [resultados, setResultados] = useState<Resultado[]>([]);
  const [carregando, setCarregando] = useState(true);
  const [erro, setErro] = useState<string>("");
  const jogo = getGame();

  useEffect(() => {
    setCarregando(true);
    getPortfolio(90)
      .then((r) => setTrades(r.trades))
      .catch(() => setErro("Não consegui carregar o estoque."))
      .finally(() => setCarregando(false));
  }, []);

  const elegiveis = useMemo(
    () =>
      (trades ?? []).filter(
        (t) => t.status === "holding" && !t.kind && t.listed && (t.askBRL ?? 0) > 0 && t.qty > 0,
      ),
    [trades],
  );

  const receber = useCallback(
    async (arquivos: FileList | null) => {
      if (!arquivos || !arquivos.length) return;
      setErro("");
      const porEdicao = new Map<string, Linha[]>();
      const infos: Catalogo["arquivos"] = [];
      let header: Linha[] = [];
      for (const f of Array.from(arquivos)) {
        const linhas = parseCSV(await f.text());
        if (linhas.length < 3) {
          setErro(`"${f.name}" não parece ser um export da Liga.`);
          return;
        }
        if (!header.length) header = linhas.slice(0, 2);
        const dados = linhas.slice(2).filter((l) => l.length > COL.preco && l[COL.edicaoId]);
        if (!dados.length) continue;
        const ed = dados[0][COL.edicaoId];
        porEdicao.set(ed, [...(porEdicao.get(ed) ?? []), ...dados]);
        infos.push({ nome: f.name, sigla: dados[0][COL.sigla], cards: dados.length });
      }
      if (!porEdicao.size) {
        setErro("Nenhuma linha de catálogo encontrada nos arquivos.");
        return;
      }
      setCatalogo({ header, porEdicao, arquivos: infos });
    },
    [],
  );

  useEffect(() => {
    if (!catalogo) return;
    setResultados(
      elegiveis.map((t) => {
        const candidatos = casar(t, catalogo);
        const cond = (t.condition || "NM").toUpperCase();
        return {
          trade: t,
          candidatos,
          escolhido: candidatos.length === 1 ? 0 : -1,
          idioma: INGLES.test(t.name || "") ? "EN" : "PT",
          qualidade: QUALIDADES.has(cond) ? cond : "NM",
        };
      }),
    );
  }, [catalogo, elegiveis]);

  const prontos = resultados.filter((r) => r.escolhido >= 0);
  const ambiguos = resultados.filter((r) => r.escolhido < 0 && r.candidatos.length > 1);
  const semMatch = resultados.filter((r) => r.candidatos.length === 0);

  const baixar = useCallback(() => {
    if (!catalogo) return;
    const linhas = prontos.map((r) => {
      const l = [...r.candidatos[r.escolhido]];
      l[COL.idioma] = r.idioma;
      l[COL.qualidade] = r.qualidade;
      l[COL.qtdSomar] = String(r.trade.qty);
      l[COL.preco] = (r.trade.askBRL ?? 0).toFixed(2).replace(".", ",");
      return l;
    });
    const escapar = (c: string) => (/[",\n]/.test(c) ? `"${c.replace(/"/g, '""')}"` : c);
    const csv = [...catalogo.header, ...linhas].map((l) => l.map(escapar).join(",")).join("\r\n");
    const url = URL.createObjectURL(new Blob(["﻿" + csv], { type: "text/csv;charset=utf-8" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = `liga-${jogo}-${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  }, [catalogo, prontos, jogo]);

  if (carregando) {
    return (
      <div className="flex items-center gap-2 text-sm text-slate-400">
        <Loader2 className="h-4 w-4 animate-spin" /> carregando estoque…
      </div>
    );
  }

  return (
    <div className="max-w-5xl space-y-6">
      <div className="sticker rounded-[12px] bg-panel p-5 text-sm leading-relaxed text-slate-200">
        Gera o CSV no formato que a Liga importa, a partir do que está{" "}
        <strong className="text-fg">à venda</strong> no Estoque. A Liga exige o <em>Carta ID</em> dela em
        cada linha, então primeiro exporte as edições que você quer em{" "}
        <strong className="text-fg">Cards e Produtos → Exportar Cards</strong> e solte os arquivos aqui —
        eu cruzo com o estoque e devolvo o arquivo pronto para{" "}
        <strong className="text-fg">Importar Cards</strong>.
      </div>

      {erro && (
        <div className="flex items-center gap-2 rounded-[10px] border-[3px] border-loss bg-loss/10 p-3 text-sm text-slate-100">
          <AlertTriangle className="h-4 w-4 text-loss" /> {erro}
        </div>
      )}

      <label className="arcade-press sticker flex cursor-pointer flex-col items-center gap-2 rounded-[12px] border-dashed bg-raised p-8 text-center">
        <FileUp className="h-6 w-6 text-brand-label" />
        <span className="font-display text-lg font-extrabold text-fg">
          Solte aqui os CSVs exportados da Liga
        </span>
        <span className="text-xs text-slate-400">um arquivo por edição — pode selecionar vários</span>
        <input
          type="file"
          accept=".csv"
          multiple
          className="hidden"
          onChange={(e) => receber(e.target.files)}
        />
      </label>

      {catalogo && (
        <div className="sticker rounded-[12px] bg-panel p-4">
          <p className="font-pixel text-[9px] text-brand-label">EDIÇÕES CARREGADAS</p>
          <div className="mt-2 flex flex-wrap gap-2">
            {catalogo.arquivos.map((a) => (
              <span key={a.nome} className="rounded-[6px] border-[2px] border-outline bg-raised px-2 py-1 text-xs text-slate-200">
                {a.sigla} · {a.cards} cards
              </span>
            ))}
          </div>
        </div>
      )}

      {!catalogo && elegiveis.length === 0 && (
        <EmptyState hint="Marque itens como à venda no Estoque para poder exportar.">Nada à venda</EmptyState>
      )}

      {catalogo && (
        <>
          <div className="grid grid-cols-3 gap-3">
            <Cartao rotulo="PRONTOS" valor={prontos.length} tom="text-gain" />
            <Cartao rotulo="AMBÍGUOS" valor={ambiguos.length} tom="text-live" />
            <Cartao rotulo="SEM MATCH" valor={semMatch.length} tom="text-loss" />
          </div>

          {ambiguos.length > 0 && (
            <div className="sticker rounded-[12px] bg-panel p-4">
              <p className="flex items-center gap-2 text-sm font-bold text-fg">
                <Info className="h-4 w-4 text-live" /> Escolha qual carta é
              </p>
              <div className="mt-3 space-y-3">
                {ambiguos.map((r) => (
                  <div key={r.trade.id} className="flex flex-wrap items-center gap-3 border-b border-slate-800 pb-3">
                    <span className="min-w-52 text-sm text-slate-200">
                      {r.trade.name} <span className="text-slate-500">· {brl(r.trade.askBRL ?? 0)}</span>
                    </span>
                    <select
                      className="rounded-[6px] border-[2px] border-outline bg-raised px-2 py-1 text-sm text-fg"
                      defaultValue=""
                      onChange={(e) =>
                        setResultados((prev) =>
                          prev.map((x) =>
                            x.trade.id === r.trade.id ? { ...x, escolhido: Number(e.target.value) } : x,
                          ),
                        )
                      }
                    >
                      <option value="" disabled>
                        {r.candidatos.length} candidatos…
                      </option>
                      {r.candidatos.map((c, i) => (
                        <option key={`${c[COL.cartaId]}-${c[COL.numero]}`} value={i}>
                          #{c[COL.numero]} · {c[COL.nomePT]} ({c[6]})
                        </option>
                      ))}
                    </select>
                  </div>
                ))}
              </div>
            </div>
          )}

          {semMatch.length > 0 && (
            <div className="sticker rounded-[12px] bg-panel p-4">
              <p className="flex items-center gap-2 text-sm font-bold text-fg">
                <X className="h-4 w-4 text-loss" /> Sem correspondência no catálogo
              </p>
              <p className="mt-1 text-xs text-slate-400">
                Faltou exportar a edição destes, ou o nome no estoque está diferente do da Liga.
              </p>
              <ul className="mt-2 space-y-1 text-sm text-slate-300">
                {semMatch.map((r) => (
                  <li key={r.trade.id}>
                    {r.trade.name} <span className="text-slate-500">· {r.trade.set}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          <div className="flex flex-wrap items-center gap-3">
            <button
              type="button"
              disabled={!prontos.length}
              onClick={baixar}
              className="arcade-press sticker sticker-sm flex items-center gap-2 rounded-[10px] bg-brand px-4 py-2 font-bold text-white disabled:opacity-40"
            >
              <Download className="h-4 w-4" />
              Baixar CSV ({prontos.length} linhas)
            </button>
            <span className="text-xs text-slate-400">
              {prontos.reduce((s, r) => s + r.trade.qty, 0)} unidades ·{" "}
              {brl(prontos.reduce((s, r) => s + r.trade.qty * (r.trade.askBRL ?? 0), 0))}
            </span>
          </div>

          <div className="sticker overflow-x-auto rounded-[12px] bg-panel">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b-[3px] border-outline bg-raised">
                  {["", "Carta", "Ed", "Nº", "Idioma", "Qual", "Qtd", "Preço"].map((h) => (
                    <th key={h} className="font-pixel px-3 py-2 text-[9px] whitespace-nowrap text-brand-label">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {resultados.map((r) => {
                  const l = r.escolhido >= 0 ? r.candidatos[r.escolhido] : undefined;
                  return (
                    <tr key={r.trade.id} className="border-b border-slate-800 last:border-0">
                      <td className="px-3 py-1.5">
                        {l ? (
                          <Check className="h-3.5 w-3.5 text-gain" />
                        ) : (
                          <X className="h-3.5 w-3.5 text-loss" />
                        )}
                      </td>
                      <td className="px-3 py-1.5 text-slate-200">{r.trade.name}</td>
                      <td className="px-3 py-1.5 text-slate-400">{l ? l[COL.sigla] : "—"}</td>
                      <td className="px-3 py-1.5 text-slate-400">{l ? l[COL.numero] : "—"}</td>
                      <td className="px-3 py-1.5 text-slate-300">{r.idioma}</td>
                      <td className="px-3 py-1.5 text-slate-300">{r.qualidade}</td>
                      <td className="px-3 py-1.5 text-slate-300">{r.trade.qty}</td>
                      <td className="px-3 py-1.5 text-slate-300">{brl(r.trade.askBRL ?? 0)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          <div className="sticker rounded-[12px] bg-panel p-4 text-xs leading-relaxed text-slate-400">
            Na Liga: <strong className="text-slate-200">Cards e Produtos → Importar Cards</strong>, escolha{" "}
            <strong className="text-slate-200">Acrescentar ao Estoque</strong> e envie o arquivo. A quantidade
            é <strong className="text-slate-200">somada</strong> à existente — importar duas vezes dobra o
            estoque. O idioma sai <strong className="text-slate-200">PT</strong>, ou{" "}
            <strong className="text-slate-200">EN</strong> quando o nome traz “(Inglês)”. Variantes (Foil,
            Reverse, Master Ball) saem em branco.
          </div>
        </>
      )}
    </div>
  );
}

function Cartao({ rotulo, valor, tom }: { rotulo: string; valor: number; tom: string }) {
  return (
    <div className="sticker rounded-[12px] bg-panel p-4">
      <p className="font-pixel text-[9px] text-slate-500">{rotulo}</p>
      <p className={`font-display mt-1 text-3xl font-extrabold ${tom}`}>{valor}</p>
    </div>
  );
}
