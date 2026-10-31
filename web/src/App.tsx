import { useCallback, useEffect, useState } from "react";
import { Truck } from "lucide-react";
import {
  getGame,
  getGames,
  getStatus,
  setGame as setApiGame,
  triggerRefresh,
  unauthorizedEvent,
  type GameInfo,
  type Status,
} from "./api";
import { brandFor, iconOf, isView, ligaLabels, scopeOf, type View } from "./brand";
import DealsPage from "./components/DealsPage";
import TrackingPage from "./components/TrackingPage";
import LigaSalesPage from "./components/LigaSalesPage";
import PedidosPage, { TokenCard } from "./components/PedidosPage";
import PurchasesPage from "./components/PurchasesPage";
import ImportsPage from "./components/ImportsPage";
import JpImportPage from "./components/JpImportPage";
import FinanceiroPage from "./components/FinanceiroPage";
import StockPage from "./components/StockPage";
import ExpensesPage from "./components/ExpensesPage";
import BuyoutPage from "./components/BuyoutPage";
import BrowsePage from "./components/BrowsePage";
import QuotePage from "./components/QuotePage";
import GuiaLigaPage from "./components/GuiaLigaPage";
import LigaExportPage from "./components/LigaExportPage";
import SyncQueuePage, { useSyncPending } from "./components/SyncQueuePage";
import PromosPage from "./components/PromosPage";
import SelectionTray from "./components/SelectionTray";
import AppSidebar from "./components/AppSidebar";
import TopBar from "./components/TopBar";
import PageHeader from "./components/PageHeader";
import { SelectionProvider } from "./selection";
import { useOrderAlerts } from "./orderAlerts";
import { SidebarInset, SidebarProvider } from "./components/ui/sidebar";
import { Button } from "./components/ui/button";

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
  vendas: {
    title: "Vendas",
    description: "O que já vendeu, da mais recente para a mais antiga — receita, custo e lucro de cada item.",
  },
  pedidos: {
    title: "Pedidos",
    description: "Pedidos pagos pelo site — endereço, frete escolhido, rastreio e baixa automática no estoque.",
  },
  compras: {
    title: "Estoque e Compras",
    description: "Tudo o que você tem e quanto pagou, de todos os jogos: lançar compra, editar item, dar baixa e mover cada item.",
  },
  importar: {
    title: "Importar / Exportar",
    description:
      "Piso em inglês da Liga contra o anúncio ao vivo da TCGplayer, nos dois sentidos — o que trazer dos EUA e o que mandar pra lá.",
  },
  importarjp: {
    title: "Importar Japão",
    description: "AR e SAR japonesas da Card Rush contra o piso japonês e português da Liga — o que vale trazer do Japão.",
  },
  financeiro: {
    title: "Financeiro",
    description: "Quanto entrou, quanto veio da Liga e o que sobrou depois das despesas.",
  },
  despesas: {
    title: "Despesas",
    description: "Gastos do negócio — lançamentos do mês e despesas fixas recorrentes.",
  },
  estoque: {
    title: "Anúncios",
    description: "Preço de venda, vitrine do site e Liga — como cada item está anunciado.",
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
    description: "Gera o CSV de importação da LigaMagic a partir do que está à venda em Anúncios.",
  },
  cupons: {
    title: "Cupons",
    description: "Códigos de desconto da vitrine: criar, limitar, pausar e ver quanto cada um rendeu.",
  },
  sincronizar: {
    title: "Sincronizar",
    description: "Saídas de estoque fora da Liga que ainda precisam ser acertadas lá.",
  },
  guialiga: {
    title: "Guia da Liga",
    description: "Como mexer na loja da LigaMagic no dia a dia — passo a passo, com prints.",
  },
};

// As telas são tabelas item a item: a largura de leitura de texto espremia as
// colunas e deixava tela vazia dos dois lados.
const pageWidth = "max-w-[1800px]";

export default function App() {
  const [view, setView] = useView();
  const [game, setGameState] = useState<string>(getGame());
  const [games, setGames] = useState<GameInfo[]>([]);
  const [status, setStatus] = useState<Status | null>(null);
  const needsToken = useUnauthorized();

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
  const orderAlerts = useOrderAlerts(activeView === "pedidos" && !needsToken);
  const syncPending = useSyncPending();
  const meta = pageMeta[activeView];
  const ViewIcon = iconOf(activeView);
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
          counts={{ pedidos: orderAlerts.toShip, sincronizar: syncPending }}
        />
        <SidebarInset>
          <TopBar game={game} status={status} onRefresh={onRefresh} dealsEnabled={dealsEnabled} />
          <div key={game} className="bg-grid flex flex-1 flex-col">
            <main
              key={activeView}
              className={`animate-fade-in mx-auto w-full flex-1 px-4 py-7 sm:px-6 lg:px-8 ${pageWidth}`}
            >
              {orderAlerts.fresh > 0 && activeView !== "pedidos" && (
                <NewOrdersBanner count={orderAlerts.fresh} onOpen={() => setView("pedidos")} />
              )}

              <PageHeader title={meta.title} description={meta.description} icon={ViewIcon} badge={scopeBadge} />

              {needsToken ? (
                <div className="mt-6">
                  <TokenCard onSaved={() => window.location.reload()} />
                </div>
              ) : activeView === "buyout" ? (
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
              ) : activeView === "vendas" ? (
                <div className="mt-6">
                  <LigaSalesPage />
                </div>
              ) : activeView === "pedidos" ? (
                <div className="mt-6">
                  <PedidosPage since={orderAlerts.since} onChanged={orderAlerts.refresh} />
                </div>
              ) : activeView === "compras" ? (
                <div className="mt-6">
                  <PurchasesPage />
                </div>
              ) : activeView === "importar" ? (
                <div className="mt-6">
                  <ImportsPage />
                </div>
              ) : activeView === "importarjp" ? (
                <div className="mt-6">
                  <JpImportPage />
                </div>
              ) : activeView === "financeiro" ? (
                <div className="mt-6">
                  <FinanceiroPage />
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
              ) : activeView === "cupons" ? (
                <div className="mt-6">
                  <PromosPage />
                </div>
              ) : activeView === "sincronizar" ? (
                <div className="mt-6">
                  <SyncQueuePage />
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

function useUnauthorized(): boolean {
  const [hit, setHit] = useState(false);
  useEffect(() => {
    const onHit = () => setHit(true);
    window.addEventListener(unauthorizedEvent, onHit);
    return () => window.removeEventListener(unauthorizedEvent, onHit);
  }, []);
  return hit;
}

const legacyViews: Record<string, { view: View; jogo?: string }> = {
  portfolio: { view: "compras" },
  allportfolio: { view: "compras" },
  acessorios: { view: "compras", jogo: "acessorios" },
};

function readView(): View {
  const params = new URLSearchParams(window.location.search);
  const value = params.get("tab") ?? "";
  const legacy = legacyViews[value];
  if (legacy) {
    params.set("tab", legacy.view);
    if (legacy.jogo) {
      params.set("jogo", legacy.jogo);
    }
    window.history.replaceState(null, "", `?${params.toString()}`);
    return legacy.view;
  }
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

function NewOrdersBanner({ count, onOpen }: { count: number; onOpen: () => void }) {
  const plural = count > 1;
  return (
    <div className="sticker sticker-sm mb-6 flex flex-wrap items-center justify-between gap-3 rounded-[12px] bg-panel px-4 py-3">
      <div className="flex items-center gap-3">
        <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full border-2 border-outline bg-brand text-white">
          <Truck className="h-[18px] w-[18px]" />
        </span>
        <div>
          <div className="font-display text-base font-extrabold text-fg">
            {count} {plural ? "pedidos pagos novos" : "pedido pago novo"} no site
          </div>
          <p className="text-xs text-slate-400">{plural ? "Estão esperando" : "Está esperando"} envio ou retirada.</p>
        </div>
      </div>
      <Button variant="primary" size="sm" onClick={onOpen}>
        Ver pedidos
      </Button>
    </div>
  );
}

function Footer({ game, hasDeals }: { game: string; hasDeals: boolean }) {
  const liga = ligaLabels[game] ?? "Liga";
  return (
    <footer
      className={`mx-auto w-full border-t-[3px] border-outline px-4 py-5 text-xs text-slate-500 sm:px-6 lg:px-8 ${pageWidth}`}
    >
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
