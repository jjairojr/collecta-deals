import type { GameInfo, Status } from "../api";
import { brandFor, navGroups, scopeOf, type View } from "../brand";
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
      <div className="sticker sticker-sm flex h-10 w-10 shrink-0 items-center justify-center rounded-[10px] bg-royal">
        <BrandIcon className="h-5 w-5 text-white" />
      </div>
      <div className={cn("min-w-0", collapsed && "lg:hidden")}>
        <span className="font-display block truncate text-lg font-extrabold leading-tight text-fg">{brand.title}</span>
        <p className="truncate text-[11px] leading-tight text-slate-500">{brand.sub}</p>
      </div>
    </div>
  );
}

// The switcher lives inside the "Jogo" group and goes dim on business-wide
// screens, where picking a game changes nothing.
function GameSwitcher({
  game,
  games,
  onChange,
  dimmed,
}: {
  game: string;
  games: GameInfo[];
  onChange: (g: string) => void;
  dimmed: boolean;
}) {
  const { collapsed } = useSidebar();
  if (games.length < 2 || collapsed) {
    return null;
  }
  return (
    <div
      title={dimmed ? "Esta tela não depende do jogo" : undefined}
      className={cn(
        "sticker sticker-sm flex gap-1 rounded-[12px] bg-page p-1 transition-opacity",
        dimmed && "opacity-40",
      )}
    >
      {games.map((g) => {
        const active = g.id === game;
        return (
          <button
            key={g.id}
            type="button"
            onClick={() => onChange(g.id)}
            title={g.name}
            className={cn(
              "font-pixel flex-1 rounded-full border-2 px-0.5 py-2 text-[9px] transition-colors",
              active ? "border-outline bg-brand text-white" : "border-transparent text-slate-400 hover:text-fg",
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
  const word = refreshing ? "Sincronizando" : ready ? "Ao vivo" : "Aguardando";

  return (
    <div className={cn("flex items-center gap-2 px-1 text-slate-500", collapsed && "lg:justify-center lg:px-0")}>
      <span className={cn("h-2 w-2 shrink-0 rounded-full border border-outline", dotClass)} title={word} />
      <span className={cn("font-pixel text-[9px] uppercase", refreshing && "text-amber-400", collapsed && "lg:hidden")}>{word}</span>
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
  const { collapsed, setMobileOpen } = useSidebar();
  const onGlobalView = scopeOf(view) === "global";

  const navigate = (next: View) => {
    onChangeView(next);
    setMobileOpen(false);
  };

  return (
    <Sidebar>
      <SidebarHeader>
        <BrandHeader game={game} />
      </SidebarHeader>

      <SidebarContent>
        {navGroups.map((group, index) => {
          const items = group.items.filter((item) => item.key !== "deals" || dealsEnabled);
          if (items.length === 0) {
            return null;
          }
          return (
            <SidebarGroup key={group.label}>
              <SidebarGroupLabel>{group.label}</SidebarGroupLabel>
              {group.hint && (
                <span className={cn("-mt-1 px-3 pb-1 text-[10px] leading-tight text-slate-600", collapsed && "lg:hidden")}>
                  {group.hint}
                </span>
              )}
              {index === 0 && (
                <div className="pb-2">
                  <GameSwitcher game={game} games={games} onChange={onChangeGame} dimmed={onGlobalView} />
                </div>
              )}
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
