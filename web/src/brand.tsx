import type { ComponentType, ReactNode } from "react";
import {
  Anchor,
  BookOpen,
  Bot,
  Boxes,
  FileText,
  HandCoins,
  Layers,
  LayoutGrid,
  LineChart,
  Package,
  PiggyBank,
  RefreshCw,
  Plane,
  Receipt,
  ShoppingBag,
  ShoppingCart,
  Sparkles,
  Store,
  Swords,
  Truck,
  Upload,
  Wallet,
  Wand2,
} from "lucide-react";

export type View =
  | "deals"
  | "browse"
  | "tracking"
  | "sealed"
  | "portfolio"
  | "allportfolio"
  | "vendas"
  | "pedidos"
  | "compras"
  | "importar"
  | "importarjp"
  | "financeiro"
  | "acessorios"
  | "despesas"
  | "estoque"
  | "orcamento"
  | "buyout"
  | "guialiga"
  | "ligaexport"
  | "sincronizar";

export type Icon = ComponentType<{ className?: string }>;

// A view is either scoped to the game picked in the switcher or business-wide
// (financeiro, vendas, despesas, acessórios): the sidebar groups them apart so
// the switcher never looks like it drives a screen it has no say over.
export type NavScope = "game" | "global";

export interface NavItem {
  key: View;
  label: string;
  icon: Icon;
}

export interface NavGroup {
  label: string;
  hint?: string;
  scope: NavScope;
  items: NavItem[];
}

export const navGroups: NavGroup[] = [
  {
    label: "Jogo",
    scope: "game",
    items: [
      { key: "deals", label: "Ofertas", icon: Sparkles },
      { key: "browse", label: "Catálogo", icon: LayoutGrid },
      { key: "tracking", label: "Mercado", icon: LineChart },
      { key: "sealed", label: "Selados", icon: Boxes },
      { key: "estoque", label: "Estoque", icon: Store },
      { key: "orcamento", label: "Orçamento", icon: FileText },
      { key: "buyout", label: "Buyout", icon: HandCoins },
    ],
  },
  {
    label: "Negócio",
    hint: "todos os jogos",
    scope: "global",
    items: [
      { key: "financeiro", label: "Financeiro", icon: PiggyBank },
      { key: "vendas", label: "Vendas", icon: ShoppingBag },
      { key: "pedidos", label: "Pedidos", icon: Truck },
      { key: "compras", label: "Compras", icon: ShoppingCart },
      { key: "sincronizar", label: "Sincronizar", icon: RefreshCw },
      { key: "importar", label: "Importar / Exportar", icon: Plane },
      { key: "importarjp", label: "Importar Japão", icon: Plane },
      { key: "despesas", label: "Despesas", icon: Receipt },
      { key: "acessorios", label: "Acessórios", icon: Package },
      { key: "allportfolio", label: "Todos os jogos", icon: Layers },
    ],
  },
  {
    label: "Liga",
    scope: "game",
    items: [
      { key: "ligaexport", label: "Exportar p/ Liga", icon: Upload },
      { key: "guialiga", label: "Guia da Liga", icon: BookOpen },
    ],
  },
];

export const navItems: NavItem[] = navGroups.flatMap((group) => group.items);

// Views that answer to a URL but are no longer in the menu. Portfólio was
// replaced by Compras, which does the same per item; the old screen stays
// reachable by link while the new one settles.
const hiddenViews: { key: View; label: string; icon: Icon; scope: NavScope }[] = [
  { key: "portfolio", label: "Portfólio", icon: Wallet, scope: "game" },
];

export function isView(value: string): value is View {
  return navItems.some((item) => item.key === value) || hiddenViews.some((v) => v.key === value);
}

export function iconOf(view: View): Icon | undefined {
  return (
    navItems.find((item) => item.key === view)?.icon ?? hiddenViews.find((v) => v.key === view)?.icon
  );
}

export function scopeOf(view: View): NavScope {
  const group = navGroups.find((g) => g.items.some((item) => item.key === view));
  if (group) {
    return group.scope;
  }
  return hiddenViews.find((v) => v.key === view)?.scope ?? "game";
}

export interface Brand {
  title: ReactNode;
  short: string;
  sub: string;
  icon: Icon;
}

export const brands: Record<string, Brand> = {
  onepiece: {
    title: (
      <>
        OP<span className="text-brand-label">Deals</span>
      </>
    ),
    short: "OP",
    sub: "One Piece TCG · buy in Brazil, sell in the US",
    icon: Anchor,
  },
  pokemon: {
    title: (
      <>
        PKM<span className="text-brand-label">Tracker</span>
      </>
    ),
    short: "PKM",
    sub: "Pokémon TCG · Liga Brazil market tracker",
    icon: Sparkles,
  },
  riftbound: {
    title: (
      <>
        RB<span className="text-brand-label">Deals</span>
      </>
    ),
    short: "RB",
    sub: "Riftbound TCG · buy in Brazil, sell in the US",
    icon: Swords,
  },
  lorcana: {
    title: (
      <>
        LOR<span className="text-brand-label">Deals</span>
      </>
    ),
    short: "LOR",
    sub: "Disney Lorcana · buy in Brazil, sell in the US",
    icon: Wand2,
  },
  gundam: {
    title: (
      <>
        GND<span className="text-brand-label">Deals</span>
      </>
    ),
    short: "GND",
    sub: "Gundam Card Game · buy in Brazil, sell in the US",
    icon: Bot,
  },
};

export const defaultBrand = brands.onepiece;

export function brandFor(game: string): Brand {
  return brands[game] ?? defaultBrand;
}

export const searchHints: Record<string, string> = {
  onepiece: "Zoro, OP01-001",
  riftbound: "Jinx, 162",
  lorcana: "Elsa, 42",
  gundam: "Gundam, GD01-001",
};

export const ligaLabels: Record<string, string> = {
  onepiece: "LigaOnePiece",
  pokemon: "LigaPokemon",
  riftbound: "LigaRiftbound",
  lorcana: "LigaLorcana",
  gundam: "LigaGundam",
};
