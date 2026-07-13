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
      <div className="min-h-screen bg-slate-950">{children}</div>
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
          "fixed inset-0 z-40 bg-black/70 backdrop-blur-sm transition-opacity duration-300 lg:hidden",
          mobileOpen ? "opacity-100" : "pointer-events-none opacity-0",
        )}
      />
      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-50 flex w-64 flex-col border-r border-slate-800 bg-slate-900/85 backdrop-blur-xl",
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
  return <div className={cn("mt-auto flex flex-col gap-2 border-t border-slate-800/80 p-3", className)} {...props} />;
}

export function SidebarGroup({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("flex flex-col gap-1", className)} {...props} />;
}

export function SidebarGroupLabel({ className, children }: { className?: string; children: ReactNode }) {
  const { collapsed } = useSidebar();
  return (
    <span
      className={cn(
        "px-3 pb-1 text-[10px] font-semibold uppercase tracking-[0.14em] text-slate-500 transition-opacity duration-200",
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
        "group/item relative flex h-10 items-center gap-3 rounded-lg px-3 text-sm font-medium outline-none transition-colors",
        "focus-visible:ring-2 focus-visible:ring-accent-500/50",
        active
          ? "bg-accent-500/12 text-accent-200"
          : "text-slate-400 hover:bg-slate-800/70 hover:text-slate-100",
        collapsed && "lg:justify-center lg:px-0",
        className,
      )}
      {...props}
    >
      <span
        aria-hidden
        className={cn(
          "absolute left-0 top-1/2 h-5 w-[3px] -translate-y-1/2 rounded-r-full bg-accent-400 transition-opacity",
          active ? "opacity-100" : "opacity-0",
        )}
      />
      <IconCmp
        className={cn(
          "h-[18px] w-[18px] shrink-0 transition-colors",
          active ? "text-accent-300" : "text-slate-500 group-hover/item:text-slate-200",
        )}
      />
      <span className={cn("flex-1 truncate text-left", collapsed && "lg:hidden")}>{label}</span>
      {badge !== undefined && badge !== null && <span className={cn(collapsed && "lg:hidden")}>{badge}</span>}
      {collapsed && (
        <span className="pointer-events-none absolute left-full top-1/2 z-50 ml-3 hidden -translate-y-1/2 whitespace-nowrap rounded-md border border-slate-700 bg-slate-800 px-2 py-1 text-xs font-medium text-slate-100 opacity-0 shadow-lg transition-opacity duration-150 group-hover/item:opacity-100 lg:block lg:group-hover/item:opacity-100">
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
        "hidden h-9 items-center gap-2 rounded-lg px-3 text-xs font-medium text-slate-400 transition-colors hover:bg-slate-800/70 hover:text-slate-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500/50 lg:flex",
        collapsed && "justify-center px-0",
        className,
      )}
    >
      <ChevronsLeft className={cn("h-4 w-4 shrink-0 transition-transform duration-300", collapsed && "rotate-180")} />
      <span className={cn(collapsed && "hidden")}>Collapse</span>
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
        "inline-flex h-9 w-9 items-center justify-center rounded-lg border border-slate-800 bg-slate-900/60 text-slate-300 transition-colors hover:bg-slate-800 hover:text-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500/50 lg:hidden",
        className,
      )}
    >
      <Menu className="h-[18px] w-[18px]" />
    </button>
  );
}
