import type { ComponentType, ReactNode } from "react";
import {
  Anchor,
  BookOpen,
  Bot,
  Boxes,
  FileText,
  HandCoins,
  LayoutGrid,
  LineChart,
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
  Wand2,
} from "lucide-react";

export type View =
  | "deals"
  | "browse"
  | "tracking"
  | "sealed"
  | "vendas"
  | "pedidos"
  | "compras"
  | "importar"
  | "importarjp"
  | "financeiro"
  | "despesas"
  | "estoque"
  | "orcamento"
  | "buyout"
  | "guialiga"
  | "ligaexport"
  | "sincronizar";

export type Icon = ComponentType<{ className?: string }>;

// A view is either scoped to the game picked in the switcher or business-wide
// (financeiro, vendas, despesas, compras): the sidebar groups them apart so
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
      { key: "estoque", label: "Anúncios", icon: Store },
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
      { key: "compras", label: "Estoque e Compras", icon: ShoppingCart },
      { key: "sincronizar", label: "Sincronizar", icon: RefreshCw },
      { key: "importar", label: "Importar / Exportar", icon: Plane },
      { key: "importarjp", label: "Importar Japão", icon: Plane },
      { key: "despesas", label: "Despesas", icon: Receipt },
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

export function isView(value: string): value is View {
  return navItems.some((item) => item.key === value);
}

export function iconOf(view: View): Icon | undefined {
  return navItems.find((item) => item.key === view)?.icon;
}

export function scopeOf(view: View): NavScope {
  return navGroups.find((g) => g.items.some((item) => item.key === view))?.scope ?? "game";
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
