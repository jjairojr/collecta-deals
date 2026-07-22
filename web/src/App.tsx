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
import { isView, ligaLabels, navItems, type View } from "./brand";
import DealsPage from "./components/DealsPage";
import TrackingPage from "./components/TrackingPage";
import PortfolioPage from "./components/PortfolioPage";
import AllPortfolioPage from "./components/AllPortfolioPage";
import BuyoutPage from "./components/BuyoutPage";
import BrowsePage from "./components/BrowsePage";
import QuotePage from "./components/QuotePage";
import SelectionTray from "./components/SelectionTray";
import AppSidebar from "./components/AppSidebar";
import TopBar from "./components/TopBar";
import PageHeader from "./components/PageHeader";
import { SelectionProvider } from "./selection";
import { SidebarInset, SidebarProvider } from "./components/ui/sidebar";

const pageMeta: Record<View, { title: string; description: string }> = {
  deals: {
    title: "Deals",
    description: "Cross-border price gaps — cheapest Brazil listing vs live US floor.",
  },
  browse: {
    title: "Browse",
    description: "Explore the full catalog with live prices.",
  },
  tracking: {
    title: "Tracking",
    description: "Daily price trends and per-store sales across the Brazil market.",
  },
  sealed: {
    title: "Sealed",
    description: "Sealed products — price trends and inferred sales.",
  },
  portfolio: {
    title: "Portfolio",
    description: "Your holdings, valuation, and realized P&L.",
  },
  allportfolio: {
    title: "All Games",
    description: "Combined portfolio across every game.",
  },
  orcamento: {
    title: "Orçamento",
    description: "Build a purchase quote to send a customer.",
  },
  buyout: {
    title: "Buyout",
    description: "Value a bulk buyout lot against the market.",
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
          <div key={game} className="flex flex-1 flex-col">
            <main key={activeView} className="animate-fade-in mx-auto w-full max-w-7xl flex-1 px-4 py-7 sm:px-6 lg:px-8">
              <PageHeader title={meta.title} description={meta.description} icon={ViewIcon} />

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
              ) : activeView === "orcamento" ? (
                <div className="mt-6">
                  <QuotePage />
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
    <footer className="mx-auto w-full max-w-7xl border-t border-slate-800/80 px-4 py-5 text-xs text-slate-500 sm:px-6 lg:px-8">
      {hasDeals ? (
        <>
          Margin is an FX-adjusted gross price gap (lowest current TCGPlayer listing vs cheapest {liga} price). It does
          not include TCGPlayer fees or shipping. High-value deals use live TCGPlayer listing prices and are verified to
          have current {liga} sellers. Catalog via TCGCSV; live prices via TCGPlayer; Brazil prices via {liga}.
        </>
      ) : (
        <>
          Brazil market data via {liga}. Prices are the current per-store floor; quantities and prices are decoded per
          snapshot. Sales are inferred from day-over-day per-store stock drops. Not affiliated with {liga}.
        </>
      )}
    </footer>
  );
}
