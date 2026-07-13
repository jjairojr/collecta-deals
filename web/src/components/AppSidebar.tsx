import type { GameInfo, Status } from "../api";
import { brandFor, navGroups, type View } from "../brand";
import { cn } from "../lib/utils";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarNavItem,
  SidebarTrigger,
  useSidebar,
} from "./ui/sidebar";

function BrandHeader({ game }: { game: string }) {
  const { collapsed } = useSidebar();
  const brand = brandFor(game);
  const BrandIcon = brand.icon;
  return (
    <div className={cn("flex items-center gap-2.5 px-1 pt-1", collapsed && "lg:justify-center lg:px-0")}>
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-accent-500/25 to-accent-500/5 ring-1 ring-inset ring-accent-500/25">
        <BrandIcon className="h-5 w-5 text-accent-300" />
      </div>
      <div className={cn("min-w-0", collapsed && "lg:hidden")}>
        <h1 className="truncate text-lg font-bold leading-tight tracking-tight text-slate-100">{brand.title}</h1>
        <p className="truncate text-[11px] leading-tight text-slate-500">{brand.sub}</p>
      </div>
    </div>
  );
}

function GameSwitcher({
  game,
  games,
  onChange,
}: {
  game: string;
  games: GameInfo[];
  onChange: (g: string) => void;
}) {
  const { collapsed } = useSidebar();
  if (games.length < 2 || collapsed) {
    return null;
  }
  return (
    <div className="grid grid-cols-4 gap-1 rounded-xl border border-slate-800 bg-slate-950/40 p-1">
      {games.map((g) => {
        const active = g.id === game;
        return (
          <button
            key={g.id}
            type="button"
            onClick={() => onChange(g.id)}
            title={g.name}
            className={cn(
              "rounded-lg px-1 py-1.5 text-[11px] font-semibold transition-colors",
              active ? "bg-accent-500/15 text-accent-200 ring-1 ring-inset ring-accent-500/30" : "text-slate-400 hover:bg-slate-800/70 hover:text-slate-200",
            )}
          >
            {brandFor(g.id).short}
          </button>
        );
      })}
    </div>
  );
}

function FooterStatus({ status }: { status: Status | null }) {
  const { collapsed } = useSidebar();
  const refreshing = status?.refreshing ?? false;
  const ready = status?.ready ?? false;
  const dotClass = refreshing ? "animate-pulse bg-amber-400" : ready ? "bg-emerald-400" : "bg-slate-600";
  const word = refreshing ? "Syncing" : ready ? "Live" : "Waiting";

  return (
    <div className={cn("flex items-center gap-2 px-1 text-[11px] text-slate-500", collapsed && "lg:justify-center lg:px-0")}>
      <span className={cn("h-2 w-2 shrink-0 rounded-full", dotClass)} title={word} />
      <span className={cn(refreshing && "text-amber-400", collapsed && "lg:hidden")}>{word}</span>
    </div>
  );
}

export default function AppSidebar({
  game,
  games,
  view,
  onChangeView,
  onChangeGame,
  dealsEnabled,
  status,
}: {
  game: string;
  games: GameInfo[];
  view: View;
  onChangeView: (v: View) => void;
  onChangeGame: (g: string) => void;
  dealsEnabled: boolean;
  status: Status | null;
}) {
  const { setMobileOpen } = useSidebar();

  const navigate = (next: View) => {
    onChangeView(next);
    setMobileOpen(false);
  };

  return (
    <Sidebar>
      <SidebarHeader>
        <BrandHeader game={game} />
        <GameSwitcher game={game} games={games} onChange={onChangeGame} />
      </SidebarHeader>

      <SidebarContent>
        {navGroups.map((group) => {
          const items = group.items.filter((item) => item.key !== "deals" || dealsEnabled);
          if (items.length === 0) {
            return null;
          }
          return (
            <SidebarGroup key={group.label}>
              <SidebarGroupLabel>{group.label}</SidebarGroupLabel>
              {items.map((item) => (
                <SidebarNavItem
                  key={item.key}
                  icon={item.icon}
                  label={item.label}
                  active={view === item.key}
                  onClick={() => navigate(item.key)}
                />
              ))}
            </SidebarGroup>
          );
        })}
      </SidebarContent>

      <SidebarFooter>
        <FooterStatus status={status} />
        <SidebarTrigger />
      </SidebarFooter>
    </Sidebar>
  );
}
