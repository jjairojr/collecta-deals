export function brl(value: number): string {
  return `R$ ${value.toFixed(2).replace(".", ",")}`;
}

export function brl0(value: number): string {
  return `R$ ${Math.round(value).toLocaleString("pt-BR")}`;
}

export function usd(value: number): string {
  return `$${value.toFixed(2)}`;
}

export function pct(value: number): string {
  const rounded = Math.round(value);
  return `${rounded > 0 ? "+" : ""}${rounded}%`;
}

export function stampLabel(s: string): string {
  const m = s.match(/^(\d{4})-(\d{2})-(\d{2})(?:T(\d{2}))?$/);
  if (!m) {
    return s;
  }
  const [, , mo, d, h] = m;
  return h ? `${mo}/${d} ${h}h` : `${mo}/${d}`;
}

export function fullStamp(iso: string): string {
  if (!iso) {
    return "";
  }
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return "";
  }
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function timeAgo(iso: string): string {
  if (!iso) {
    return "never";
  }
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) {
    return "never";
  }
  const seconds = Math.floor((Date.now() - then) / 1000);
  if (seconds < 60) {
    return "just now";
  }
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return `${minutes}m ago`;
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return `${hours}h ago`;
  }
  return `${Math.floor(hours / 24)}d ago`;
}
