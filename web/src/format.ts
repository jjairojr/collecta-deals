export function brl(value: number): string {
  return `R$ ${value.toFixed(2).replace(".", ",")}`;
}

export function brl0(value: number): string {
  return `R$ ${Math.round(value).toLocaleString("pt-BR")}`;
}

export function brl2(value: number): string {
  return value.toLocaleString("pt-BR", { style: "currency", currency: "BRL" });
}

export function usd(value: number): string {
  return `$${value.toFixed(2)}`;
}

export function pct(value: number): string {
  const rounded = Math.round(value);
  return `${rounded > 0 ? "+" : ""}${rounded}%`;
}

// dayLabel renders an ISO date the Brazilian way (DD/MM), for ledgers and charts
// whose dates are read by the store owner, not by the US side of the business.
export function dayLabel(date: string): string {
  const m = date.match(/^\d{4}-(\d{2})-(\d{2})$/);
  return m ? `${m[2]}/${m[1]}` : date;
}

export function stampLabel(s: string): string {
  const m = s.match(/^(\d{4})-(\d{2})-(\d{2})(?:T(\d{2}))?$/);
  if (!m) {
    return s;
  }
  const [, , mo, d, h] = m;
  return h ? `${d}/${mo} ${h}h` : `${d}/${mo}`;
}

export function fullStamp(iso: string): string {
  if (!iso) {
    return "";
  }
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return "";
  }
  return d.toLocaleString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function timeAgo(iso: string): string {
  if (!iso) {
    return "nunca";
  }
  const then = new Date(iso).getTime();
  if (Number.isNaN(then)) {
    return "nunca";
  }
  const seconds = Math.floor((Date.now() - then) / 1000);
  if (seconds < 60) {
    return "agora";
  }
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return `há ${minutes}min`;
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return `há ${hours}h`;
  }
  return `há ${Math.floor(hours / 24)}d`;
}
