import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ButtonHTMLAttributes,
  type HTMLAttributes,
  type ReactNode,
} from "react";
import { ChevronsLeft, Menu } from "lucide-react";
import { cn } from "../../lib/utils";
import type { Icon } from "../../brand";

const STORAGE_KEY = "opdeals.sidebar.collapsed";
const EXPANDED = "lg:w-64";
const COLLAPSED = "lg:w-[4.25rem]";
const INSET_EXPANDED = "lg:pl-64";
const INSET_COLLAPSED = "lg:pl-[4.25rem]";

interface SidebarContextValue {
  collapsed: boolean;
  toggleCollapsed: () => void;
  mobileOpen: boolean;
  setMobileOpen: (open: boolean) => void;
}

const SidebarContext = createContext<SidebarContextValue | null>(null);

export function useSidebar(): SidebarContextValue {
  const ctx = useContext(SidebarContext);
  if (!ctx) {
    throw new Error("useSidebar must be used within <SidebarProvider>");
  }
  return ctx;
}

function readCollapsed(): boolean {
  try {
    return window.localStorage.getItem(STORAGE_KEY) === "1";
  } catch {
    return false;
  }
}

export function SidebarProvider({ children }: { children: ReactNode }) {
  const [collapsed, setCollapsed] = useState<boolean>(readCollapsed);
  const [mobileOpen, setMobileOpen] = useState(false);

  const toggleCollapsed = useCallback(() => {
    setCollapsed((prev) => {
      const next = !prev;
      try {
        window.localStorage.setItem(STORAGE_KEY, next ? "1" : "0");
      } catch {
        void 0;
      }
      return next;
    });
  }, []);

  useEffect(() => {
    if (!mobileOpen) {
      return;
    }
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setMobileOpen(false);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [mobileOpen]);

  const value = useMemo(
    () => ({ collapsed, toggleCollapsed, mobileOpen, setMobileOpen }),
    [collapsed, toggleCollapsed, mobileOpen],
  );

  return (
    <SidebarContext.Provider value={value}>
      <div className="min-h-screen bg-page">{children}</div>
    </SidebarContext.Provider>
  );
}

export function Sidebar({ className, children }: { className?: string; children: ReactNode }) {
  const { collapsed, mobileOpen, setMobileOpen } = useSidebar();
  return (
    <>
      <div
        onClick={() => setMobileOpen(false)}
        aria-hidden
        className={cn(
          "fixed inset-0 z-40 bg-outline/60 backdrop-blur-sm transition-opacity duration-300 lg:hidden",
          mobileOpen ? "opacity-100" : "pointer-events-none opacity-0",
        )}
      />
      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-r-4 border-outline bg-panel",
          "transition-[transform,width] duration-300 ease-[cubic-bezier(0.22,1,0.36,1)]",
          collapsed ? COLLAPSED : EXPANDED,
          mobileOpen ? "translate-x-0" : "-translate-x-full",
          "lg:translate-x-0",
          className,
        )}
      >
        {children}
      </aside>
    </>
  );
}

export function SidebarInset({ className, children }: { className?: string; children: ReactNode }) {
  const { collapsed } = useSidebar();
  return (
    <div
      className={cn(
        "flex min-h-screen flex-col transition-[padding] duration-300 ease-[cubic-bezier(0.22,1,0.36,1)]",
        collapsed ? INSET_COLLAPSED : INSET_EXPANDED,
        className,
      )}
    >
      {children}
    </div>
  );
}

export function SidebarHeader({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("flex flex-col gap-3 p-3", className)} {...props} />;
}

export function SidebarContent({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("flex flex-1 flex-col gap-5 overflow-y-auto overflow-x-hidden px-3 py-4", className)}
      {...props}
    />
  );
}

export function SidebarFooter({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("mt-auto flex flex-col gap-2 border-t-[3px] border-outline p-3", className)} {...props} />;
}

export function SidebarGroup({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("flex flex-col gap-1", className)} {...props} />;
}

export function SidebarGroupLabel({ className, children }: { className?: string; children: ReactNode }) {
  const { collapsed } = useSidebar();
  return (
    <span
      className={cn(
        "font-pixel block px-3 pb-1.5 text-[9px] uppercase leading-[1.7] tracking-normal text-brand-label transition-opacity duration-200",
        collapsed ? "pointer-events-none h-0 select-none overflow-hidden pb-0 opacity-0 lg:h-0" : "opacity-100",
        className,
      )}
    >
      {children}
    </span>
  );
}

interface SidebarNavItemProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  icon: Icon;
  label: string;
  active?: boolean;
  badge?: ReactNode;
}

export function SidebarNavItem({ icon: IconCmp, label, active, badge, className, ...props }: SidebarNavItemProps) {
  const { collapsed } = useSidebar();
  return (
    <button
      type="button"
      aria-current={active ? "page" : undefined}
      className={cn(
        "group/item relative flex h-10 items-center gap-3 rounded-full border-2 px-3 text-sm font-semibold outline-none transition-colors",
        active
          ? "border-outline bg-brand text-white shadow-[3px_3px_0_var(--color-outline)]"
          : "border-transparent text-slate-400 hover:border-outline hover:bg-raised",
        collapsed && "lg:justify-center lg:px-0",
        className,
      )}
      {...props}
    >
      <IconCmp
        className={cn(
          "h-[18px] w-[18px] shrink-0 transition-colors",
          active ? "text-fg" : "text-slate-500 group-hover/item:text-fg",
        )}
      />
      <span className={cn("flex-1 truncate text-left", collapsed && "lg:hidden")}>{label}</span>
      {badge !== undefined && badge !== null && <span className={cn(collapsed && "lg:hidden")}>{badge}</span>}
      {collapsed && (
        <span className="pointer-events-none absolute left-full top-1/2 z-50 ml-3 hidden -translate-y-1/2 whitespace-nowrap rounded-[8px] border-2 border-outline bg-raised px-2 py-1 text-xs font-medium text-white opacity-0 sticker sticker-sm transition-opacity duration-150 group-hover/item:opacity-100 lg:block lg:group-hover/item:opacity-100">
          {label}
        </span>
      )}
    </button>
  );
}

export function SidebarTrigger({ className }: { className?: string }) {
  const { collapsed, toggleCollapsed } = useSidebar();
  return (
    <button
      type="button"
      onClick={toggleCollapsed}
      aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
      className={cn(
        "font-pixel hidden h-9 items-center gap-2 rounded-full border-2 border-transparent px-3 text-[9px] uppercase text-slate-400 transition-colors hover:border-outline hover:bg-raised hover:text-fg lg:flex",
        collapsed && "justify-center px-0",
        className,
      )}
    >
      <ChevronsLeft className={cn("h-4 w-4 shrink-0 transition-transform duration-300", collapsed && "rotate-180")} />
      <span className={cn(collapsed && "hidden")}>Recolher</span>
    </button>
  );
}

export function SidebarMobileTrigger({ className }: { className?: string }) {
  const { setMobileOpen } = useSidebar();
  return (
    <button
      type="button"
      onClick={() => setMobileOpen(true)}
      aria-label="Open navigation"
      className={cn(
        "pill pill-sm arcade-press inline-flex h-9 w-9 items-center justify-center bg-panel text-fg transition-colors hover:bg-raised lg:hidden",
        className,
      )}
    >
      <Menu className="h-[18px] w-[18px]" />
    </button>
  );
}
