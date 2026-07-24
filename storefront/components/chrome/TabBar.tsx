"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronDown } from "lucide-react";
import Container from "@/components/ui/Container";
import { gamePixel } from "@/lib/games";

interface NavGame {
  id: string;
  name: string;
}

// HOME + one tab per game (hover reveals a Singles/Selados popover) + CARRINHO.
// Product pages (/carta, /selado/[slug]) highlight no tab; the browsing context
// is reachable from the breadcrumb.
export default function TabBar({ games }: { games: NavGame[] }) {
  const pathname = usePathname();
  return (
    <nav className="relative z-30 border-b-4 border-royal bg-outline">
      <Container className="flex items-stretch">
        <div className="flex min-w-0 flex-1 items-stretch">
          <TabLink href="/" label="HOME" active={pathname === "/"} />
          {games.map((g) => (
            <GameMenu key={g.id} game={g} />
          ))}
          <TabLink
            href="/carrinho"
            label="CARRINHO"
            active={pathname.startsWith("/carrinho")}
          />
        </div>
        <span className="hidden shrink-0 items-center px-5 font-pixel text-[10px] text-brand animate-blink sm:flex">
          INSERT COIN
        </span>
      </Container>
    </nav>
  );
}

function TabLink({
  href,
  label,
  active,
}: {
  href: string;
  label: string;
  active: boolean;
}) {
  return (
    <Link
      href={href}
      aria-current={active ? "page" : undefined}
      className={`flex items-center whitespace-nowrap border-r-2 border-ink px-4 py-4 font-pixel text-[10px] transition-colors sm:px-5 ${
        active ? "bg-brand text-white" : "text-[#8a8a92] hover:text-white"
      }`}
    >
      {label}
    </Link>
  );
}

// A game tab. Hovering (or keyboard-focusing) opens a popover to pick Singles or
// Selados for that game. The tab label itself links to the game's singles as a
// default — so it still works on touch, where there is no hover.
function GameMenu({ game }: { game: NavGame }) {
  return (
    <div className="group relative flex items-stretch">
      <Link
        href={`/singles?jogo=${game.id}`}
        onClick={(e) => e.currentTarget.blur()}
        className="flex items-center gap-1.5 whitespace-nowrap border-r-2 border-ink px-4 py-4 font-pixel text-[10px] text-[#8a8a92] transition-colors group-hover:bg-brand group-hover:text-white group-focus-within:bg-brand group-focus-within:text-white sm:px-5"
      >
        {gamePixel(game.id, game.name)}
        <ChevronDown className="h-3 w-3 transition-transform group-hover:rotate-180" />
      </Link>

      <div className="invisible absolute left-0 top-full min-w-[160px] translate-y-1 opacity-0 transition-all duration-150 group-hover:visible group-hover:translate-y-0 group-hover:opacity-100 group-focus-within:visible group-focus-within:translate-y-0 group-focus-within:opacity-100">
        <div
          role="menu"
          className="sticker mt-1 overflow-hidden rounded-[10px] bg-surface"
          style={{ ["--sh" as string]: "5px" }}
        >
          <MenuLink href={`/singles?jogo=${game.id}`} label="SINGLES" />
          <MenuLink href={`/selado?jogo=${game.id}`} label="SELADOS" />
        </div>
      </div>
    </div>
  );
}

function MenuLink({ href, label }: { href: string; label: string }) {
  return (
    <Link
      role="menuitem"
      href={href}
      onClick={(e) => e.currentTarget.blur()}
      className="block whitespace-nowrap border-b-2 border-outline px-4 py-3 font-pixel text-[9px] text-[#c9c9d1] transition-colors last:border-b-0 hover:bg-brand hover:text-white"
    >
      {label}
    </Link>
  );
}
