"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import Container from "@/components/ui/Container";
import Field from "@/components/checkout/Field";
import UFSelect from "@/components/checkout/UFSelect";
import { trackBeginCheckout, trackGenerateLead } from "@/lib/analytics";
import { useCart } from "@/lib/cart";
import {
  cents,
  digits,
  lookupCEP,
  maskCEP,
  maskCPF,
  maskPhone,
  orderLines,
  validateAddress,
  validateBuyer,
  type Address,
  type Buyer,
  type CheckoutError,
  type ShippingOption,
} from "@/lib/checkout";
import { brl, pixelText } from "@/lib/format";
import { buildOrder, whatsappURL } from "@/lib/whatsapp";

const EMPTY_BUYER: Buyer = { name: "", email: "", phone: "", cpf: "" };
const EMPTY_ADDRESS: Address = {
  cep: "",
  street: "",
  number: "",
  complement: "",
  district: "",
  city: "",
  state: "",
};

type BuyerErrors = Partial<Record<keyof Buyer, string>>;
type AddressErrors = Partial<Record<keyof Address, string>>;

interface CreateResponse {
  id: string;
  initPoint: string;
}

export default function CheckoutView() {
  const { lines, total: itemsCents, reprice, cap } = useCart();
  const [buyer, setBuyer] = useState<Buyer>(EMPTY_BUYER);
  const [address, setAddress] = useState<Address>(EMPTY_ADDRESS);
  const [buyerErrors, setBuyerErrors] = useState<BuyerErrors>({});
  const [addressErrors, setAddressErrors] = useState<AddressErrors>({});
  const [cepLoading, setCepLoading] = useState(false);
  const [options, setOptions] = useState<ShippingOption[] | null>(null);
  const [quoting, setQuoting] = useState(false);
  const [quoteError, setQuoteError] = useState<string | null>(null);
  const [serviceID, setServiceID] = useState<number | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const begun = useRef(false);

  useEffect(() => {
    if (!begun.current && lines.length > 0) {
      begun.current = true;
      trackBeginCheckout(lines, itemsCents);
    }
  }, [lines, itemsCents]);

  const cepDigits = digits(address.cep);
  const payloadLines = useMemo(() => orderLines(lines), [lines]);

  useEffect(() => {
    if (cepDigits.length !== 8 || payloadLines.length === 0) {
      setOptions(null);
      setServiceID(null);
      return;
    }
    let cancelled = false;
    setQuoting(true);
    setQuoteError(null);
    fetch("/api/frete", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ cep: cepDigits, lines: payloadLines }),
    })
      .then(async (res) => {
        const data = (await res.json()) as { services?: ShippingOption[]; error?: string };
        if (!res.ok) {
          throw new Error(data.error ?? "frete indisponível");
        }
        return data.services ?? [];
      })
      .then((services) => {
        if (cancelled) {
          return;
        }
        setOptions(services);
        setServiceID((prev) =>
          services.some((s) => s.serviceID === prev)
            ? prev
            : (services[0]?.serviceID ?? null),
        );
      })
      .catch((err: unknown) => {
        if (cancelled) {
          return;
        }
        setOptions([]);
        setQuoteError(err instanceof Error ? err.message : "frete indisponível");
      })
      .finally(() => {
        if (!cancelled) {
          setQuoting(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [cepDigits, payloadLines]);

  const chosen = options?.find((s) => s.serviceID === serviceID) ?? null;
  const shippingCents = chosen ? cents(chosen.priceBRL) : 0;
  const totalCents = itemsCents + shippingCents;

  if (lines.length === 0) {
    return (
      <Container narrow className="py-16">
        <div
          className="sticker sticker-5 mx-auto flex max-w-md flex-col items-center gap-5 rounded-[18px] border-brand bg-outline px-8 py-14 text-center"
          style={{ ["--sh" as string]: "8px" }}
        >
          <h1 className="font-display text-4xl font-bold text-white">
            Nada para pagar
          </h1>
          <p className="font-pixel text-[10px] leading-relaxed text-brand-soft">
            SEU CARRINHO ESTA VAZIO
          </p>
          <Link
            href="/singles"
            className="arcade-press sticker rounded-[10px] bg-brand px-5 py-3 font-pixel text-[10px] text-white"
            style={{ ["--sh" as string]: "5px" }}
          >
            EXPLORAR SINGLES
          </Link>
        </div>
      </Container>
    );
  }

  const setBuyerField = (key: keyof Buyer, value: string) => {
    setBuyer((b) => ({ ...b, [key]: value }));
    if (buyerErrors[key]) {
      setBuyerErrors((e) => ({ ...e, [key]: undefined }));
    }
  };

  const setAddressField = (key: keyof Address, value: string) => {
    setAddress((a) => ({ ...a, [key]: value }));
    if (addressErrors[key]) {
      setAddressErrors((e) => ({ ...e, [key]: undefined }));
    }
  };

  const onCEP = async (raw: string) => {
    const masked = maskCEP(raw);
    setAddressField("cep", masked);
    if (digits(masked).length !== 8) {
      return;
    }
    setCepLoading(true);
    const found = await lookupCEP(masked);
    setCepLoading(false);
    if (!found) {
      return;
    }
    setAddress((a) => ({
      ...a,
      cep: masked,
      street: found.street || a.street,
      district: found.district || a.district,
      city: found.city || a.city,
      state: found.state || a.state,
    }));
    setAddressErrors((e) => ({ ...e, street: undefined, district: undefined, city: undefined, state: undefined }));
  };

  const submit = async () => {
    const be = validateBuyer(buyer);
    const ae = validateAddress(address);
    setBuyerErrors(be);
    setAddressErrors(ae);
    if (Object.keys(be).length > 0 || Object.keys(ae).length > 0) {
      setNotice("Confira os campos destacados.");
      return;
    }
    if (!chosen) {
      setNotice("Escolha uma opção de frete.");
      return;
    }
    setSubmitting(true);
    setNotice(null);
    try {
      const res = await fetch("/api/pedido", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          lines: payloadLines,
          buyer,
          address: { ...address, cep: cepDigits },
          shipping: { serviceID: chosen.serviceID, priceBRL: chosen.priceBRL },
        }),
      });
      const data = (await res.json()) as CreateResponse & CheckoutError;
      if (res.ok && data.initPoint) {
        trackGenerateLead(totalCents, "mercadopago");
        window.location.assign(data.initPoint);
        return;
      }
      if (data.error === "price_changed" && data.lines) {
        const prices: Record<string, number> = {};
        for (const l of data.lines) {
          if (typeof l.unitCents === "number") {
            prices[l.sku] = l.unitCents;
          }
        }
        reprice(prices);
        setNotice("Alguns preços mudaram desde que você montou o carrinho. Confira o total e tente de novo.");
      } else if (data.error === "out_of_stock" && data.lines) {
        const stock: Record<string, number> = {};
        for (const l of data.lines) {
          stock[l.sku] = l.available;
        }
        cap(stock);
        setNotice("Parte do estoque acabou enquanto você finalizava. Ajustamos as quantidades do carrinho.");
      } else if (data.error === "shipping_changed" && data.services) {
        setOptions(data.services);
        setServiceID(data.services[0]?.serviceID ?? null);
        setNotice("O valor do frete mudou. Escolha de novo a forma de envio.");
      } else {
        setNotice(data.error ?? "Não foi possível criar o pedido. Tente de novo em instantes.");
      }
    } catch {
      setNotice("Sem conexão com a loja. Tente de novo em instantes.");
    } finally {
      setSubmitting(false);
    }
  };

  const whatsapp = () => {
    trackGenerateLead(itemsCents);
    window.open(whatsappURL(buildOrder(lines, itemsCents)), "_blank", "noopener");
  };

  return (
    <Container narrow className="py-9">
      <h1 className="font-display text-4xl font-bold leading-none text-white sm:text-[52px]">
        Checkout
      </h1>
      <div className="mt-3 font-pixel text-[9px] text-brand">
        PLAYER 1 · PREENCHA E PAGUE
      </div>

      <div className="mt-7 grid gap-7 lg:grid-cols-[1.4fr_.6fr] lg:items-start">
        <div className="min-w-0 space-y-5">
          <Section step="01" title="Seus dados">
            <div className="grid gap-4 sm:grid-cols-2">
              <Field
                label="Nome completo"
                value={buyer.name}
                onChange={(v) => setBuyerField("name", v)}
                error={buyerErrors.name}
                autoComplete="name"
                className="sm:col-span-2"
              />
              <Field
                label="E-mail"
                value={buyer.email}
                onChange={(v) => setBuyerField("email", v)}
                error={buyerErrors.email}
                inputMode="email"
                autoComplete="email"
              />
              <Field
                label="Celular / WhatsApp"
                value={buyer.phone}
                onChange={(v) => setBuyerField("phone", maskPhone(v))}
                error={buyerErrors.phone}
                inputMode="tel"
                autoComplete="tel-national"
                placeholder="(11) 99999-9999"
              />
              <Field
                label="CPF"
                value={buyer.cpf}
                onChange={(v) => setBuyerField("cpf", maskCPF(v))}
                error={buyerErrors.cpf}
                inputMode="numeric"
                placeholder="000.000.000-00"
              />
            </div>
          </Section>

          <Section step="02" title="Endereço de entrega">
            <div className="grid gap-4 sm:grid-cols-6">
              <Field
                label={cepLoading ? "CEP · buscando" : "CEP"}
                value={address.cep}
                onChange={(v) => void onCEP(v)}
                error={addressErrors.cep}
                inputMode="numeric"
                autoComplete="postal-code"
                placeholder="00000-000"
                className="sm:col-span-2"
              />
              <Field
                label="Rua"
                value={address.street}
                onChange={(v) => setAddressField("street", v)}
                error={addressErrors.street}
                autoComplete="address-line1"
                className="sm:col-span-4"
              />
              <Field
                label="Número"
                value={address.number}
                onChange={(v) => setAddressField("number", v)}
                error={addressErrors.number}
                className="sm:col-span-2"
              />
              <Field
                label="Complemento"
                value={address.complement}
                onChange={(v) => setAddressField("complement", v)}
                placeholder="apto, bloco…"
                className="sm:col-span-4"
              />
              <Field
                label="Bairro"
                value={address.district}
                onChange={(v) => setAddressField("district", v)}
                error={addressErrors.district}
                className="sm:col-span-2"
              />
              <Field
                label="Cidade"
                value={address.city}
                onChange={(v) => setAddressField("city", v)}
                error={addressErrors.city}
                autoComplete="address-level2"
                className="sm:col-span-3"
              />
              <div className="sm:col-span-1">
                <UFSelect
                  value={address.state}
                  onChange={(v) => setAddressField("state", v)}
                  error={addressErrors.state}
                />
              </div>
            </div>
          </Section>

          <Section step="03" title="Frete">
            {cepDigits.length !== 8 ? (
              <p className="text-sm text-muted">
                Informe o CEP para ver as opções de envio.
              </p>
            ) : quoting ? (
              <p className="font-pixel text-[9px] text-brand-soft animate-blink">
                CALCULANDO FRETE...
              </p>
            ) : quoteError ? (
              <p className="text-sm text-brand">
                Não conseguimos cotar o frete agora ({quoteError}). Tente de novo ou
                feche pelo WhatsApp.
              </p>
            ) : options && options.length === 0 ? (
              <p className="text-sm text-brand">
                Nenhuma transportadora atende esse CEP com esse pacote. Fale com a
                gente no WhatsApp.
              </p>
            ) : (
              <div className="grid gap-3">
                {options?.map((s) => (
                  <FreightOption
                    key={s.serviceID}
                    option={s}
                    selected={s.serviceID === serviceID}
                    onSelect={() => setServiceID(s.serviceID)}
                  />
                ))}
              </div>
            )}
          </Section>
        </div>

        <div
          className="sticker sticker-5 min-w-0 rounded-[16px] border-brand bg-outline p-5 sm:p-[26px] lg:sticky lg:top-24"
          style={{ ["--sh" as string]: "8px" }}
        >
          <div className="font-pixel text-[11px] text-brand-soft">RESUMO</div>

          <ul className="mt-4 space-y-2 text-sm">
            {lines.map((l) => (
              <li
                key={`${l.sku}|${l.meta}`}
                className="flex items-start justify-between gap-3"
              >
                <span className="min-w-0">
                  <span className="block truncate text-white">
                    {l.qty}× {l.name}
                  </span>
                  <span className="block truncate text-xs text-muted">{l.meta}</span>
                </span>
                <span className="shrink-0 font-pixel text-[10px] text-white">
                  {brl(l.price * l.qty)}
                </span>
              </li>
            ))}
          </ul>

          <dl className="mt-5 space-y-3 border-t-[3px] border-brand pt-5 text-[15px]">
            <Row label="Subtotal" value={brl(itemsCents)} />
            <Row
              label={chosen ? `Frete · ${chosen.name}` : "Frete"}
              value={chosen ? (shippingCents === 0 ? "Grátis" : brl(shippingCents)) : "—"}
            />
          </dl>

          <div className="my-5 border-t-[3px] border-brand" />

          <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
            <span className="font-pixel text-[10px] text-white">TOTAL</span>
            <span className="font-pixel text-[18px] text-brand sm:text-[22px]">
              {brl(totalCents)}
            </span>
          </div>
          <p className="mt-2 text-xs text-muted">Pix ou cartão em até 12x</p>

          {notice && (
            <p
              role="alert"
              className="sticker mt-4 rounded-[10px] bg-brand-soft px-3 py-2 text-xs font-semibold text-outline [--sh:3px]"
            >
              {notice}
            </p>
          )}

          <button
            type="button"
            onClick={() => void submit()}
            disabled={submitting || quoting}
            className="mt-5 w-full rounded-[10px] border-4 border-white bg-brand px-6 py-3.5 font-pixel text-[11px] text-white transition-transform hover:-translate-y-0.5 disabled:cursor-wait disabled:opacity-60"
            style={{ boxShadow: "0 0 0 4px #0b0b0c" }}
          >
            {submitting ? "ABRINDO PAGAMENTO..." : "PAGAR COM MERCADO PAGO"}
          </button>
          <button
            type="button"
            onClick={whatsapp}
            className="mt-4 block w-full rounded px-2 py-2 text-center text-xs text-muted underline-offset-4 hover:text-brand-soft hover:underline"
          >
            ou feche o pedido pelo WhatsApp
          </button>
          <p className="mt-3 text-center font-pixel text-[8px] leading-relaxed text-faint">
            VOCE SERA REDIRECIONADO AO MERCADO PAGO
            <br />
            ESTOQUE RESERVADO POR 30 MINUTOS
          </p>
        </div>
      </div>
    </Container>
  );
}

function Section({
  step,
  title,
  children,
}: {
  step: string;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <section className="sticker rounded-[14px] bg-surface p-5 sm:p-6 [--sh:5px]">
      <div className="mb-4 flex items-baseline gap-3">
        <span className="font-pixel text-[9px] text-brand">{step}</span>
        <h2 className="font-display text-2xl font-bold leading-none text-white">
          {title}
        </h2>
      </div>
      {children}
    </section>
  );
}

function FreightOption({
  option,
  selected,
  onSelect,
}: {
  option: ShippingOption;
  selected: boolean;
  onSelect: () => void;
}) {
  const price = option.priceBRL === 0 ? "GRATIS" : brl(cents(option.priceBRL));
  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      onClick={onSelect}
      className={`sticker flex w-full items-center justify-between gap-3 rounded-[10px] px-4 py-3 text-left transition-colors [--sh:3px] ${
        selected ? "bg-brand text-white" : "bg-outline text-[#c9c9d1] hover:text-white"
      }`}
    >
      <span className="min-w-0">
        <span className="block truncate font-pixel text-[8px]">
          {pixelText(`${option.company} · ${option.name}`)}
        </span>
        <span className={`block text-xs ${selected ? "text-white/90" : "text-muted"}`}>
          até {option.deliveryDays} {option.deliveryDays === 1 ? "dia útil" : "dias úteis"}
        </span>
      </span>
      <span className="shrink-0 font-pixel text-[11px]">{price}</span>
    </button>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <dt className="truncate text-[#c9c9d1]">{label}</dt>
      <dd className="shrink-0 text-white">{value}</dd>
    </div>
  );
}
