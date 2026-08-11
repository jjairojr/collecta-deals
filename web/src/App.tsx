import { useCallback, useEffect, useState } from "react";
import {
  getGame,
  getGames,
  getStatus,
  setGame as setApiGame,
  triggerRefresh,
  type GameInfo,
  type Status,
} from "./api";
import { brandFor, isView, ligaLabels, navItems, scopeOf, type View } from "./brand";
import DealsPage from "./components/DealsPage";
import TrackingPage from "./components/TrackingPage";
import PortfolioPage from "./components/PortfolioPage";
import AllPortfolioPage from "./components/AllPortfolioPage";
import LigaSalesPage from "./components/LigaSalesPage";
import FinanceiroPage from "./components/FinanceiroPage";
import StockPage from "./components/StockPage";
import ExpensesPage from "./components/ExpensesPage";
import BuyoutPage from "./components/BuyoutPage";
import BrowsePage from "./components/BrowsePage";
import QuotePage from "./components/QuotePage";
import GuiaLigaPage from "./components/GuiaLigaPage";
import LigaExportPage from "./components/LigaExportPage";
import SelectionTray from "./components/SelectionTray";
import AppSidebar from "./components/AppSidebar";
import TopBar from "./components/TopBar";
import PageHeader from "./components/PageHeader";
import { SelectionProvider } from "./selection";
import { SidebarInset, SidebarProvider } from "./components/ui/sidebar";

const pageMeta: Record<View, { title: string; description: string }> = {
  deals: {
    title: "Ofertas",
    description: "Diferença de preço entre o piso da Liga e o mercado americano.",
  },
  browse: {
    title: "Catálogo",
    description: "Todas as cartas do jogo com preço atual.",
  },
  tracking: {
    title: "Mercado",
    description: "Preço dia a dia e vendas por loja no mercado brasileiro.",
  },
  sealed: {
    title: "Selados",
    description: "Produtos selados — tendência de preço e vendas inferidas.",
  },
  portfolio: {
    title: "Portfólio",
    description: "Suas compras e vendas — quanto investiu e quanto já vendeu.",
  },
  allportfolio: {
    title: "Todos os jogos",
    description: "Portfólio somado de todos os jogos.",
  },
  vendas: {
    title: "Vendas da Liga",
    description: "Pedidos já vendidos — receita, custo e lucro de cada pacote, em todos os jogos.",
  },
  financeiro: {
    title: "Financeiro",
    description: "Quanto entrou, quanto veio da Liga e o que sobrou depois das despesas.",
  },
  acessorios: {
    title: "Acessórios",
    description: "Sleeves, deckboxes, playmats — estoque único, o mesmo em todos os jogos.",
  },
  despesas: {
    title: "Despesas",
    description: "Gastos do negócio — lançamentos do mês e despesas fixas recorrentes.",
  },
  estoque: {
    title: "Estoque",
    description: "Defina preços e escolha o que aparece na vitrine pública.",
  },
  orcamento: {
    title: "Orçamento",
    description: "Monte um orçamento de compra para mandar pro cliente.",
  },
  buyout: {
    title: "Buyout",
    description: "Avalie um lote inteiro contra o mercado.",
  },
  ligaexport: {
    title: "Exportar p/ Liga",
    description: "Gera o CSV de importação da LigaMagic a partir do que está à venda no Estoque.",
  },
  guialiga: {
    title: "Guia da Liga",
    description: "Como mexer na loja da LigaMagic no dia a dia — passo a passo, com prints.",
  },
};

function iconFor(view: View) {
  return navItems.find((item) => item.key === view)?.icon;
}

export default function App() {
  const [view, setView] = useView();
  const [game, setGameState] = useState<string>(getGame());
  const [games, setGames] = useState<GameInfo[]>([]);
  const [status, setStatus] = useState<Status | null>(null);

  const refreshStatus = useCallback(async () => {
    try {
      setStatus(await getStatus());
    } catch {
      setStatus(null);
    }
  }, []);

  useEffect(() => {
    refreshStatus();
    const id = window.setInterval(refreshStatus, 5000);
    return () => window.clearInterval(id);
  }, [refreshStatus, game]);

  useEffect(() => {
    getGames()
      .then((data) => setGames(data.games))
      .catch(() => setGames([]));
  }, []);

  const activeGame = games.find((g) => g.id === game);
  const dealsEnabled = activeGame ? activeGame.hasDeals : game !== "pokemon";

  const changeGame = useCallback(
    (next: string) => {
      if (next === game) {
        return;
      }
      setApiGame(next);
      setGameState(next);
      setStatus(null);
      const params = new URLSearchParams(window.location.search);
      if (next === "onepiece") {
        params.delete("game");
      } else {
        params.set("game", next);
      }
      const q = params.toString();
      window.history.replaceState(null, "", q ? `?${q}` : window.location.pathname);
    },
    [game],
  );

  useEffect(() => {
    if (!dealsEnabled && view === "deals") {
      setView("tracking");
    }
  }, [dealsEnabled, view, setView]);

  const onRefresh = useCallback(async () => {
    await triggerRefresh();
    refreshStatus();
  }, [refreshStatus]);

  const activeView: View = view === "deals" && !dealsEnabled ? "tracking" : view;
  const meta = pageMeta[activeView];
  const ViewIcon = iconFor(activeView);
  const scopeBadge = (
    <span className="pill pill-sm bg-panel px-2.5 py-1 font-pixel text-[8px] uppercase text-brand-label">
      {scopeOf(activeView) === "global" ? "Todos os jogos" : brandFor(game).short}
    </span>
  );

  return (
    <SelectionProvider>
      <SidebarProvider>
        <AppSidebar
          game={game}
          games={games}
          view={activeView}
          onChangeView={setView}
          onChangeGame={changeGame}
          dealsEnabled={dealsEnabled}
          status={status}
        />
        <SidebarInset>
          <TopBar game={game} status={status} onRefresh={onRefresh} dealsEnabled={dealsEnabled} />
          <div key={game} className="bg-grid flex flex-1 flex-col">
            <main key={activeView} className="animate-fade-in mx-auto w-full max-w-7xl flex-1 px-4 py-7 sm:px-6 lg:px-8">
              <PageHeader title={meta.title} description={meta.description} icon={ViewIcon} badge={scopeBadge} />

              {activeView === "buyout" ? (
                <div className="mt-6">
                  <BuyoutPage />
                </div>
              ) : activeView === "browse" ? (
                <div className="mt-6">
                  <BrowsePage />
                </div>
              ) : activeView === "tracking" ? (
                <div className="mt-6">
                  <TrackingPage key="singles" mode="singles" />
                </div>
              ) : activeView === "sealed" ? (
                <div className="mt-6">
                  <TrackingPage key="sealed" mode="sealed" />
                </div>
              ) : activeView === "portfolio" ? (
                <div className="mt-6">
                  <PortfolioPage />
                </div>
              ) : activeView === "allportfolio" ? (
                <div className="mt-6">
                  <AllPortfolioPage
                    onOpenGame={(id) => {
                      changeGame(id);
                      setView("portfolio");
                    }}
                  />
                </div>
              ) : activeView === "vendas" ? (
                <div className="mt-6">
                  <LigaSalesPage />
                </div>
              ) : activeView === "financeiro" ? (
                <div className="mt-6">
                  <FinanceiroPage />
                </div>
              ) : activeView === "acessorios" ? (
                <div className="mt-6">
                  <PortfolioPage lockedSection="accessories" />
                </div>
              ) : activeView === "despesas" ? (
                <div className="mt-6">
                  <ExpensesPage />
                </div>
              ) : activeView === "estoque" ? (
                <div className="mt-6">
                  <StockPage />
                </div>
              ) : activeView === "orcamento" ? (
                <div className="mt-6">
                  <QuotePage />
                </div>
              ) : activeView === "ligaexport" ? (
                <div className="mt-6">
                  <LigaExportPage />
                </div>
              ) : activeView === "guialiga" ? (
                <div className="mt-6">
                  <GuiaLigaPage />
                </div>
              ) : (
                <DealsPage game={game} status={status} hasMyP={activeGame?.hasMyP ?? false} />
              )}
            </main>
            <Footer game={game} hasDeals={dealsEnabled} />
          </div>
        </SidebarInset>
      </SidebarProvider>
      <SelectionTray />
    </SelectionProvider>
  );
}

function readView(): View {
  const value = new URLSearchParams(window.location.search).get("tab") ?? "";
  return isView(value) ? value : "deals";
}

function useView(): [View, (v: View) => void] {
  const [view, setViewState] = useState<View>(readView);

  useEffect(() => {
    const onPop = () => setViewState(readView());
    window.addEventListener("popstate", onPop);
    return () => window.removeEventListener("popstate", onPop);
  }, []);

  const setView = useCallback((next: View) => {
    const params = new URLSearchParams(window.location.search);
    if (next === "deals") {
      params.delete("tab");
    } else {
      params.set("tab", next);
    }
    const query = params.toString();
    window.history.pushState(null, "", query ? `?${query}` : window.location.pathname);
    setViewState(next);
  }, []);

  return [view, setView];
}

function Footer({ game, hasDeals }: { game: string; hasDeals: boolean }) {
  const liga = ligaLabels[game] ?? "Liga";
  return (
    <footer className="mx-auto w-full max-w-7xl border-t-[3px] border-outline px-4 py-5 text-xs text-slate-500 sm:px-6 lg:px-8">
      {hasDeals ? (
        <>
          A margem é a diferença bruta de preço já convertida pelo câmbio (menor anúncio atual na TCGplayer vs menor
          preço na {liga}). Não desconta taxas da TCGplayer nem frete. Ofertas de valor alto usam preço ao vivo da
          TCGplayer e são verificadas contra vendedores atuais da {liga}. Catálogo via TCGCSV; preços ao vivo via
          TCGplayer; preços do Brasil via {liga}.
        </>
      ) : (
        <>
          Dados do mercado brasileiro via {liga}. Os preços são o piso atual por loja; preços e quantidades são
          decodificados a cada snapshot. As vendas são inferidas pela queda de estoque de um dia para o outro. Sem
          vínculo com a {liga}.
        </>
      )}
    </footer>
  );
}
