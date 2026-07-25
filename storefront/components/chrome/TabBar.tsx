"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { ChevronDown } from "lucide-react";
import Container from "@/components/ui/Container";
import { gamePixel } from "@/lib/games";

interface NavGame {
  id: string;
  name: string;
}

export default function TabBar({ games }: { games: NavGame[] }) {
  const pathname = usePathname();
  const scrollerRef = useRef<HTMLDivElement | null>(null);
  const [fade, setFade] = useState({ left: false, right: false });

  useEffect(() => {
    const el = scrollerRef.current;
    if (!el) {
      return;
    }
    const update = () => {
      setFade({
        left: el.scrollLeft > 4,
        right: el.scrollLeft + el.clientWidth < el.scrollWidth - 4,
      });
    };
    update();
    el.addEventListener("scroll", update, { passive: true });
    window.addEventListener("resize", update);
    return () => {
      el.removeEventListener("scroll", update);
      window.removeEventListener("resize", update);
    };
  }, []);

  return (
    <nav className="relative z-30 border-b-4 border-royal bg-outline">
      <Container className="relative flex items-stretch">
        <div
          ref={scrollerRef}
          className="flex min-w-0 flex-1 items-stretch overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden sm:overflow-x-visible"
        >
          <TabLink href="/" label="HOME" active={pathname === "/"} />
          {games.map((g) => (
            <GameMenu key={g.id} game={g} />
          ))}
        </div>
        {fade.left && (
          <div
            aria-hidden
            className="pointer-events-none absolute inset-y-0 left-0 w-10 bg-gradient-to-r from-outline to-transparent sm:hidden"
          />
        )}
        {fade.right && (
          <div
            aria-hidden
            className="pointer-events-none absolute inset-y-0 right-0 flex w-14 items-center justify-end bg-gradient-to-l from-outline to-transparent pr-1 sm:hidden"
          >
            <span className="font-pixel text-[10px] text-brand animate-blink">
              ›
            </span>
          </div>
        )}
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

function GameMenu({ game }: { game: NavGame }) {
  const [menu, setMenu] = useState<{ left: number; top: number } | null>(null);
  const rootRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!menu) {
      return;
    }
    const close = (e: Event) => {
      if (e.target instanceof Node && rootRef.current?.contains(e.target)) {
        return;
      }
      setMenu(null);
    };
    document.addEventListener("pointerdown", close);
    document.addEventListener("scroll", close, true);
    window.addEventListener("resize", close);
    return () => {
      document.removeEventListener("pointerdown", close);
      document.removeEventListener("scroll", close, true);
      window.removeEventListener("resize", close);
    };
  }, [menu]);

  const onTabTap = (e: React.MouseEvent<HTMLAnchorElement>) => {
    if (window.matchMedia("(min-width: 640px)").matches) {
      e.currentTarget.blur();
      return;
    }
    e.preventDefault();
    if (menu) {
      setMenu(null);
      return;
    }
    const r = e.currentTarget.getBoundingClientRect();
    setMenu({
      left: Math.max(8, Math.min(r.left, window.innerWidth - 196)),
      top: r.bottom + 4,
    });
  };

  return (
    <div ref={rootRef} className="group relative flex items-stretch">
      <Link
        href={`/singles?jogo=${game.id}`}
        aria-haspopup="menu"
        aria-expanded={menu !== null}
        onClick={onTabTap}
        className="flex items-center gap-1.5 whitespace-nowrap border-r-2 border-ink px-4 py-4 font-pixel text-[10px] text-[#8a8a92] transition-colors group-hover:bg-brand group-hover:text-white group-focus-within:bg-brand group-focus-within:text-white sm:px-5"
      >
        {gamePixel(game.id, game.name)}
        <ChevronDown
          className={`h-3 w-3 transition-transform group-hover:rotate-180 ${
            menu ? "rotate-180" : ""
          }`}
        />
      </Link>

      <div className="invisible absolute left-0 top-full hidden min-w-[160px] translate-y-1 opacity-0 transition-all duration-150 group-hover:visible group-hover:translate-y-0 group-hover:opacity-100 group-focus-within:visible group-focus-within:translate-y-0 group-focus-within:opacity-100 sm:block">
        <div
          role="menu"
          className="sticker mt-1 overflow-hidden rounded-[10px] bg-surface [--sh:5px]"
        >
          <MenuLink href={`/singles?jogo=${game.id}`} label="SINGLES" />
          <MenuLink href={`/selado?jogo=${game.id}`} label="SELADOS" />
        </div>
      </div>

      {menu && (
        <div
          className="fixed z-50 sm:hidden"
          style={{ left: menu.left, top: menu.top }}
        >
          <div
            role="menu"
            className="sticker w-[188px] overflow-hidden rounded-[10px] bg-surface [--sh:5px]"
          >
            <MenuLink
              href={`/singles?jogo=${game.id}`}
              label="SINGLES"
              onNavigate={() => setMenu(null)}
            />
            <MenuLink
              href={`/selado?jogo=${game.id}`}
              label="SELADOS"
              onNavigate={() => setMenu(null)}
            />
          </div>
        </div>
      )}
    </div>
  );
}

function MenuLink({
  href,
  label,
  onNavigate,
}: {
  href: string;
  label: string;
  onNavigate?: () => void;
}) {
  return (
    <Link
      role="menuitem"
      href={href}
      onClick={(e) => {
        e.currentTarget.blur();
        onNavigate?.();
      }}
      className="block whitespace-nowrap border-b-2 border-outline px-4 py-3.5 font-pixel text-[9px] text-[#c9c9d1] transition-colors last:border-b-0 hover:bg-brand hover:text-white"
    >
      {label}
    </Link>
  );
}
