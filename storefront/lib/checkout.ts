import type { CartLine, ProductKind } from "@/lib/types";

export type OrderStatus =
  | "pending"
  | "paid"
  | "failed"
  | "expired"
  | "cancelled"
  | "oversold"
  | "shipped"
  | "refunded";

export const PICKUP_SERVICE_ID = -1;
export const JOIN_SERVICE_ID = -2;

export function isPickup(serviceID: number): boolean {
  return serviceID === PICKUP_SERVICE_ID;
}

export function isJoin(serviceID: number): boolean {
  return serviceID === JOIN_SERVICE_ID;
}

export interface ShippingOption {
  serviceID: number;
  name: string;
  company: string;
  priceBRL: number;
  deliveryDays: number;
}

export interface Buyer {
  name: string;
  email: string;
  phone: string;
  cpf: string;
}

export interface Address {
  cep: string;
  street: string;
  number: string;
  complement: string;
  district: string;
  city: string;
  state: string;
}

export interface OrderLineInput {
  sku: string;
  qty: number;
  unitCents: number;
}

export interface PublicOrderLine {
  name: string;
  meta?: string;
  kind: ProductKind;
  qty: number;
  unitBRL: number;
}

export interface PublicOrder {
  id: string;
  number: number;
  status: OrderStatus;
  lines: PublicOrderLine[];
  totals: { itemsBRL: number; shippingBRL: number; totalBRL: number };
  shipping: {
    serviceID: number;
    serviceName: string;
    company: string;
    priceBRL: number;
    deliveryDays: number;
  };
  trackingCode?: string;
  initPoint?: string;
  expiresAt: string;
  firstName: string;
}

export interface LineProblem {
  sku: string;
  unitCents?: number;
  available: number;
}

export interface CheckoutError {
  error: string;
  lines?: LineProblem[];
  services?: ShippingOption[];
}

export const UFS = [
  "AC", "AL", "AP", "AM", "BA", "CE", "DF", "ES", "GO", "MA", "MT", "MS", "MG",
  "PA", "PB", "PR", "PE", "PI", "RJ", "RN", "RS", "RO", "RR", "SC", "SP", "SE", "TO",
];

export function digits(s: string): string {
  return s.replace(/\D/g, "");
}

export function maskCPF(s: string): string {
  const d = digits(s).slice(0, 11);
  return d
    .replace(/^(\d{3})(\d)/, "$1.$2")
    .replace(/^(\d{3})\.(\d{3})(\d)/, "$1.$2.$3")
    .replace(/\.(\d{3})(\d{1,2})$/, ".$1-$2");
}

export function maskCEP(s: string): string {
  const d = digits(s).slice(0, 8);
  return d.length > 5 ? `${d.slice(0, 5)}-${d.slice(5)}` : d;
}

export function maskPhone(s: string): string {
  const d = digits(s).slice(0, 11);
  if (d.length <= 2) {
    return d;
  }
  if (d.length <= 6) {
    return `(${d.slice(0, 2)}) ${d.slice(2)}`;
  }
  if (d.length <= 10) {
    return `(${d.slice(0, 2)}) ${d.slice(2, 6)}-${d.slice(6)}`;
  }
  return `(${d.slice(0, 2)}) ${d.slice(2, 7)}-${d.slice(7)}`;
}

export function validCPF(s: string): boolean {
  const d = digits(s);
  if (d.length !== 11 || /^(\d)\1{10}$/.test(d)) {
    return false;
  }
  const check = (n: number) => {
    let sum = 0;
    for (let i = 0; i < n; i++) {
      sum += Number(d[i]) * (n + 1 - i);
    }
    const r = (sum * 10) % 11;
    return r === 10 ? 0 : r;
  };
  return check(9) === Number(d[9]) && check(10) === Number(d[10]);
}

export function validEmail(s: string): boolean {
  return /^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(s.trim());
}

export function validateBuyer(b: Buyer): Partial<Record<keyof Buyer, string>> {
  const errors: Partial<Record<keyof Buyer, string>> = {};
  if (b.name.trim().split(/\s+/).filter(Boolean).length < 2) {
    errors.name = "Nome e sobrenome";
  }
  if (!validEmail(b.email)) {
    errors.email = "E-mail inválido";
  }
  const phone = digits(b.phone);
  if (phone.length < 10 || phone.length > 11) {
    errors.phone = "DDD + número";
  }
  if (!validCPF(b.cpf)) {
    errors.cpf = "CPF inválido";
  }
  return errors;
}

export function validateAddress(a: Address): Partial<Record<keyof Address, string>> {
  const errors: Partial<Record<keyof Address, string>> = {};
  if (digits(a.cep).length !== 8) {
    errors.cep = "CEP com 8 dígitos";
  }
  if (!a.street.trim()) {
    errors.street = "Obrigatório";
  }
  if (!a.number.trim()) {
    errors.number = "Obrigatório";
  }
  if (!a.district.trim()) {
    errors.district = "Obrigatório";
  }
  if (!a.city.trim()) {
    errors.city = "Obrigatório";
  }
  if (!UFS.includes(a.state)) {
    errors.state = "UF";
  }
  return errors;
}

export interface CEPResult {
  street: string;
  district: string;
  city: string;
  state: string;
}

interface ViaCEP {
  erro?: boolean;
  logradouro?: string;
  bairro?: string;
  localidade?: string;
  uf?: string;
}

export async function lookupCEP(cep: string): Promise<CEPResult | null> {
  const d = digits(cep);
  if (d.length !== 8) {
    return null;
  }
  try {
    const res = await fetch(`https://viacep.com.br/ws/${d}/json/`);
    if (!res.ok) {
      return null;
    }
    const data = (await res.json()) as ViaCEP;
    if (data.erro) {
      return null;
    }
    return {
      street: data.logradouro ?? "",
      district: data.bairro ?? "",
      city: data.localidade ?? "",
      state: data.uf ?? "",
    };
  } catch {
    return null;
  }
}

export function orderLines(lines: CartLine[]): OrderLineInput[] {
  return lines.map((l) => ({ sku: l.sku, qty: l.qty, unitCents: l.price }));
}

export function cents(brl: number): number {
  return Math.round(brl * 100);
}

export const ORDER_STORAGE_PREFIX = "collecta.purchase.";
