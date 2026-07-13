import type { ComponentType, ReactNode } from "react";
import {
  Anchor,
  Bot,
  Boxes,
  FileText,
  HandCoins,
  Layers,
  LayoutGrid,
  LineChart,
  Sparkles,
  Swords,
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
  | "orcamento"
  | "buyout";

export type Icon = ComponentType<{ className?: string }>;

export interface NavItem {
  key: View;
  label: string;
  icon: Icon;
}

export interface NavGroup {
  label: string;
  items: NavItem[];
}

export const navGroups: NavGroup[] = [
  {
    label: "Discover",
    items: [
      { key: "deals", label: "Deals", icon: Sparkles },
      { key: "browse", label: "Browse", icon: LayoutGrid },
    ],
  },
  {
    label: "Market",
    items: [
      { key: "tracking", label: "Tracking", icon: LineChart },
      { key: "sealed", label: "Sealed", icon: Boxes },
    ],
  },
  {
    label: "Inventory",
    items: [
      { key: "portfolio", label: "Portfolio", icon: Wallet },
      { key: "allportfolio", label: "All Games", icon: Layers },
      { key: "orcamento", label: "Orçamento", icon: FileText },
      { key: "buyout", label: "Buyout", icon: HandCoins },
    ],
  },
];

export const navItems: NavItem[] = navGroups.flatMap((group) => group.items);

export function isView(value: string): value is View {
  return navItems.some((item) => item.key === value);
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
        OP<span className="text-accent-400">Deals</span>
      </>
    ),
    short: "OP",
    sub: "One Piece TCG · buy in Brazil, sell in the US",
    icon: Anchor,
  },
  pokemon: {
    title: (
      <>
        PKM<span className="text-accent-400">Tracker</span>
      </>
    ),
    short: "PKM",
    sub: "Pokémon TCG · Liga Brazil market tracker",
    icon: Sparkles,
  },
  riftbound: {
    title: (
      <>
        RB<span className="text-accent-400">Deals</span>
      </>
    ),
    short: "RB",
    sub: "Riftbound TCG · buy in Brazil, sell in the US",
    icon: Swords,
  },
  lorcana: {
    title: (
      <>
        LOR<span className="text-accent-400">Deals</span>
      </>
    ),
    short: "LOR",
    sub: "Disney Lorcana · buy in Brazil, sell in the US",
    icon: Wand2,
  },
  gundam: {
    title: (
      <>
        GND<span className="text-accent-400">Deals</span>
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
