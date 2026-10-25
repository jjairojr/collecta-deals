"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import Container from "@/components/ui/Container";
import Field from "@/components/checkout/Field";
import UFSelect from "@/components/checkout/UFSelect";
import { useAccount } from "@/lib/account";
import {
  trackApplyCoupon,
  trackBeginCheckout,
  trackCouponError,
  trackGenerateLead,
  trackLoginPrompt,
  trackRemoveCoupon,
} from "@/lib/analytics";
import { useCart } from "@/lib/cart";
import {
  cents,
  digits,
  isJoin,
  isPickup,
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
  type OrderLineInput,
  type ShippingOption,
} from "@/lib/checkout";
import { couponsApplicable, promoDiscountCents, type CheckoutCoupon, type PromoCoupon } from "@/lib/coupons";
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

type Mode = "ship" | "pickup" | "join";
type BuyerErrors = Partial<Record<keyof Buyer, string>>;
type AddressErrors = Partial<Record<keyof Address, string>>;

interface CreateResponse {
  id: string;
  initPoint: string;
}

async function loadCoupons(): Promise<CheckoutCoupon[]> {
  const res = await fetch("/api/conta/cupons", { cache: "no-store" });
  if (!res.ok) {
    return [];
  }
  const data = (await res.json()) as { coupons?: CheckoutCoupon[] };
  return data.coupons ?? [];
}

async function checkPromo(code: string): Promise<PromoCoupon> {
  const res = await fetch("/api/cupom", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ code }),
  });
  const data = (await res.json()) as Partial<PromoCoupon> & { error?: string; message?: string };
  if (res.ok && data.code && typeof data.percentBps === "number") {
    return { code: data.code, percentBps: data.percentBps };
  }
  if (data.error === "login_required") {
    throw new Error("Sua sessão expirou. Entre de novo para usar cupons.");
  }
  throw new Error(data.message ?? "Esse cupom não pôde ser aplicado.");
}

async function quote(cep: string, lines: OrderLineInput[]): Promise<ShippingOption[]> {
  const res = await fetch("/api/frete", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ cep, lines }),
  });
  const data = (await res.json()) as { services?: ShippingOption[]; error?: string };
  if (!res.ok) {
    throw new Error(data.error ?? "frete indisponível");
  }
  return data.services ?? [];
}

export default function CheckoutView() {
  const { lines, total: itemsCents, reprice, cap } = useCart();
  const { status: accountStatus, user } = useAccount();
  const [buyer, setBuyer] = useState<Buyer>(EMPTY_BUYER);
  const [coupons, setCoupons] = useState<CheckoutCoupon[] | null>(null);
  const [selectedCoupons, setSelectedCoupons] = useState<string[]>([]);
  const [couponsVersion, setCouponsVersion] = useState(0);
  const [promo, setPromo] = useState<PromoCoupon | null>(null);
  const [promoInput, setPromoInput] = useState("");
  const [promoError, setPromoError] = useState<string | null>(null);
  const [promoChecking, setPromoChecking] = useState(false);
  const [address, setAddress] = useState<Address>(EMPTY_ADDRESS);
  const [buyerErrors, setBuyerErrors] = useState<BuyerErrors>({});
  const [addressErrors, setAddressErrors] = useState<AddressErrors>({});
  const [cepLoading, setCepLoading] = useState(false);
  const [mode, setMode] = useState<Mode>("ship");
  const [local, setLocal] = useState<ShippingOption[]>([]);
  const [carriers, setCarriers] = useState<ShippingOption[] | null>(null);
  const [quoting, setQuoting] = useState(false);
  const [quoteError, setQuoteError] = useState<string | null>(null);
  const [carrierID, setCarrierID] = useState<number | null>(null);
  const [joinOrder, setJoinOrder] = useState("");
  const [joinError, setJoinError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const begun = useRef(false);

  useEffect(() => {
    if (!begun.current && lines.length > 0) {
      begun.current = true;
      trackBeginCheckout(lines, itemsCents);
    }
  }, [lines, itemsCents]);

  useEffect(() => {
    document.body.classList.add("no-scanlines");
    return () => document.body.classList.remove("no-scanlines");
  }, []);

  useEffect(() => {
    if (accountStatus !== "ready" || !user) {
      return;
    }
    setBuyer((b) => ({
      ...b,
      name: b.name || user.displayName,
      email: b.email || (user.email ?? ""),
      phone: b.phone || (user.whatsapp ? maskPhone(user.whatsapp) : ""),
    }));
  }, [accountStatus, user]);

  useEffect(() => {
    if (accountStatus !== "ready") {
      setCoupons(null);
      setSelectedCoupons([]);
      setPromo(null);
      return;
    }
    let cancelled = false;
    loadCoupons()
      .then((found) => {
        if (!cancelled) {
          setCoupons(found);
          setSelectedCoupons((prev) => prev.filter((id) => found.some((c) => c.id === id)));
        }
      })
      .catch(() => {
        if (!cancelled) {
          setCoupons([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [accountStatus, couponsVersion]);

  const cepDigits = digits(address.cep);
  const payloadLines = useMemo(() => orderLines(lines), [lines]);

  useEffect(() => {
    if (payloadLines.length === 0) {
      return;
    }
    let cancelled = false;
    quote("", payloadLines)
      .then((services) => {
        if (!cancelled) {
          setLocal(services.filter((s) => s.serviceID < 0));
        }
      })
      .catch(() => {
        if (!cancelled) {
          setLocal([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [payloadLines]);

  useEffect(() => {
    if (mode !== "ship" || cepDigits.length !== 8 || payloadLines.length === 0) {
      setCarriers(null);
      setCarrierID(null);
      return;
    }
    let cancelled = false;
    setQuoting(true);
    setQuoteError(null);
    quote(cepDigits, payloadLines)
      .then((services) => {
        if (cancelled) {
          return;
        }
        const found = services.filter((s) => s.serviceID >= 0);
        setCarriers(found);
        setCarrierID((prev) =>
          found.some((s) => s.serviceID === prev) ? prev : (found[0]?.serviceID ?? null),
        );
      })
      .catch((err: unknown) => {
        if (cancelled) {
          return;
        }
        setCarriers([]);
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
  }, [mode, cepDigits, payloadLines]);

  const pickupOption = local.find((s) => isPickup(s.serviceID)) ?? null;
  const joinOption = local.find((s) => isJoin(s.serviceID)) ?? null;
  const chosen: ShippingOption | null =
    mode === "pickup"
      ? pickupOption
      : mode === "join"
        ? joinOption
        : (carriers?.find((s) => s.serviceID === carrierID) ?? null);
  const shippingCents = chosen ? cents(chosen.priceBRL) : 0;
  const minOrderCents = coupons?.[0]?.minOrderCents;
  const applicable = couponsApplicable(coupons?.length ?? 0, itemsCents, minOrderCents);
  const activeCoupons = (coupons ?? []).filter((c) => selectedCoupons.includes(c.id)).slice(0, applicable);
  const promoCents = promo ? promoDiscountCents(itemsCents, promo.percentBps) : 0;
  const discountCents = promo ? promoCents : activeCoupons.reduce((sum, c) => sum + c.valueCents, 0);
  const discountCodes = promo ? [promo.code] : activeCoupons.map((c) => c.code);
  const totalCents = Math.max(itemsCents + shippingCents - discountCents, 0);

  const toggleCoupon = (c: CheckoutCoupon) => {
    setSelectedCoupons((prev) => {
      if (prev.includes(c.id)) {
        trackRemoveCoupon(c.code);
        return prev.filter((id) => id !== c.id);
      }
      if (prev.length >= applicable) {
        return prev;
      }
      if (promo) {
        trackRemoveCoupon(promo.code);
        setPromo(null);
      }
      trackApplyCoupon(c.code, c.valueCents);
      return [...prev, c.id];
    });
  };

  const applyPromo = async () => {
    const code = promoInput.trim();
    if (!code) {
      setPromoError("Digite o código");
      return;
    }
    setPromoChecking(true);
    setPromoError(null);
    try {
      const found = await checkPromo(code);
      setPromo(found);
      setPromoInput("");
      setSelectedCoupons([]);
      trackApplyCoupon(found.code, promoDiscountCents(itemsCents, found.percentBps));
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Esse cupom não pôde ser aplicado.";
      trackCouponError(message);
      setPromoError(message);
    } finally {
      setPromoChecking(false);
    }
  };

  const removePromo = () => {
    if (promo) {
      trackRemoveCoupon(promo.code);
    }
    setPromo(null);
  };

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
          {notice && <p className="text-sm text-[#c9c9d1]">{notice}</p>}
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
    const ae = mode === "ship" ? validateAddress(address) : {};
    setBuyerErrors(be);
    setAddressErrors(ae);
    if (Object.keys(be).length > 0 || Object.keys(ae).length > 0) {
      setNotice("Confira os campos destacados.");
      return;
    }
    if (!chosen) {
      setNotice(mode === "ship" ? "Escolha uma opção de frete." : "Escolha como quer receber.");
      return;
    }
    if (mode === "join" && !/^\d+$/.test(joinOrder.trim())) {
      setJoinError("Número do pedido");
      setNotice("Informe o número do pedido pago que vai junto.");
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
          address: mode === "ship" ? { ...address, cep: cepDigits } : EMPTY_ADDRESS,
          shipping: {
            serviceID: chosen.serviceID,
            priceBRL: chosen.priceBRL,
            joinOrder: mode === "join" ? Number(joinOrder) : 0,
          },
          couponIds: promo ? [] : activeCoupons.map((c) => c.id),
          promoCode: promo?.code ?? "",
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
      } else if (data.error === "join_not_found") {
        setJoinError("Não encontrado");
        setNotice("Não achamos um pedido pago com esse número no seu CPF ou e-mail, ou ele já foi enviado.");
      } else if (data.error === "shipping_changed" && data.services) {
        const found = data.services.filter((s) => s.serviceID >= 0);
        setCarriers(found);
        setCarrierID(found[0]?.serviceID ?? null);
        setNotice("O valor do frete mudou. Escolha de novo a forma de envio.");
      } else if (data.error === "coupon_invalid") {
        trackCouponError(data.message ?? "coupon_invalid");
        setSelectedCoupons([]);
        setPromo(null);
        setCouponsVersion((n) => n + 1);
        setNotice(data.message ?? "Esse cupom não pôde ser aplicado. Confira seus cupons e tente de novo.");
      } else if (data.error === "login_required") {
        trackCouponError("login_required");
        setSelectedCoupons([]);
        setPromo(null);
        setNotice("Sua sessão expirou. Entre de novo para usar cupons.");
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
    window.open(whatsappURL(buildOrder(lines, itemsCents, promo ? [{ code: promo.code, valueCents: promoCents }] : activeCoupons)), "_blank", "noopener");
  };

  const modes: { key: Mode; title: string; detail: string; price: string }[] = [
    { key: "ship", title: "Receber pelos Correios", detail: "frete calculado pelo seu CEP", price: "" },
  ];
  if (pickupOption) {
    modes.push({
      key: "pickup",
      title: pickupOption.name,
      detail: `${pickupOption.company} · combinar pelo WhatsApp`,
      price: "GRATIS",
    });
  }
  if (joinOption) {
    modes.push({
      key: "join",
      title: joinOption.name,
      detail: "na mesma encomenda de um pedido pago que ainda não saiu",
      price: "GRATIS",
    });
  }

  return (
    <Container narrow className="py-9">
      <h1 className="font-display text-4xl font-bold leading-none text-white sm:text-[52px]">
        Checkout
      </h1>
      <div className="mt-3 font-pixel text-[9px] text-brand">
        PLAYER 1 · PREENCHA E PAGUE
      </div>

      <form
        className="mt-7 grid gap-7 lg:grid-cols-[1.4fr_.6fr] lg:items-start"
        autoComplete="on"
        onSubmit={(e) => {
          e.preventDefault();
          void submit();
        }}
      >
        <div className="min-w-0 space-y-5">
          <Section step="01" title="Seus dados">
            <div className="grid gap-4 sm:grid-cols-2">
              <Field
                label="Nome completo"
                name="name"
                value={buyer.name}
                onChange={(v) => setBuyerField("name", v)}
                error={buyerErrors.name}
                autoComplete="name"
                className="sm:col-span-2"
              />
              <Field
                label="E-mail"
                name="email"
                value={buyer.email}
                onChange={(v) => setBuyerField("email", v)}
                error={buyerErrors.email}
                inputMode="email"
                autoComplete="email"
              />
              <Field
                label="Celular / WhatsApp"
                name="phone"
                value={buyer.phone}
                onChange={(v) => setBuyerField("phone", maskPhone(v))}
                error={buyerErrors.phone}
                inputMode="tel"
                autoComplete="tel-national"
                placeholder="(11) 99999-9999"
              />
              <Field
                label="CPF"
                name="cpf"
                value={buyer.cpf}
                onChange={(v) => setBuyerField("cpf", maskCPF(v))}
                error={buyerErrors.cpf}
                inputMode="numeric"
                placeholder="000.000.000-00"
              />
            </div>
          </Section>

          <Section step="02" title="Como receber">
            <div className="grid gap-3">
              {modes.map((m) => (
                <OptionChip
                  key={m.key}
                  title={m.title}
                  detail={m.detail}
                  price={m.price}
                  selected={mode === m.key}
                  onSelect={() => setMode(m.key)}
                />
              ))}
            </div>

            {mode === "pickup" && (
              <p className="mt-4 text-sm text-muted">
                Depois do pagamento a gente combina dia e local da retirada pelo
                WhatsApp. Sem endereço, sem frete.
              </p>
            )}

            {mode === "join" && (
              <div className="mt-4 grid gap-2 sm:max-w-xs">
                <Field
                  label="Número do pedido pago"
                  name="joinOrder"
                  value={joinOrder}
                  onChange={(v) => {
                    setJoinOrder(digits(v));
                    setJoinError(null);
                  }}
                  error={joinError ?? undefined}
                  inputMode="numeric"
                  placeholder="ex. 12"
                />
                <p className="text-xs text-muted">
                  Só vale para pedido já pago e ainda não enviado, no mesmo CPF ou
                  e-mail. Vai para o mesmo endereço, numa encomenda só.
                </p>
              </div>
            )}
          </Section>

          {mode === "ship" && (
            <Section step="03" title="Endereço de entrega">
              <div className="grid gap-4 sm:grid-cols-6">
                <Field
                  label={cepLoading ? "CEP · buscando" : "CEP"}
                  name="cep"
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
                  name="street"
                  value={address.street}
                  onChange={(v) => setAddressField("street", v)}
                  error={addressErrors.street}
                  autoComplete="address-line1"
                  className="sm:col-span-4"
                />
                <Field
                  label="Número"
                  name="number"
                  value={address.number}
                  onChange={(v) => setAddressField("number", v)}
                  error={addressErrors.number}
                  autoComplete="address-line2"
                  className="sm:col-span-2"
                />
                <Field
                  label="Complemento"
                  name="complement"
                  value={address.complement}
                  onChange={(v) => setAddressField("complement", v)}
                  placeholder="apto, bloco…"
                  className="sm:col-span-4"
                />
                <Field
                  label="Bairro"
                  name="district"
                  value={address.district}
                  onChange={(v) => setAddressField("district", v)}
                  error={addressErrors.district}
                  autoComplete="address-level3"
                  className="sm:col-span-2"
                />
                <Field
                  label="Cidade"
                  name="city"
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

              <div className="mt-6 border-t-[3px] border-outline pt-5">
                <h3 className="mb-3 font-pixel text-[9px] text-brand-soft">FRETE</h3>
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
                    Não conseguimos cotar o frete agora ({quoteError}). Tente de novo
                    ou feche pelo WhatsApp.
                  </p>
                ) : carriers && carriers.length === 0 ? (
                  <p className="text-sm text-brand">
                    Nenhuma transportadora atende esse CEP com esse pacote. Fale com a
                    gente no WhatsApp.
                  </p>
                ) : (
                  <div className="grid gap-3">
                    {carriers?.map((s) => (
                      <OptionChip
                        key={s.serviceID}
                        title={`${s.company} · ${s.name}`}
                        detail={`até ${s.deliveryDays} ${s.deliveryDays === 1 ? "dia útil" : "dias úteis"}`}
                        price={s.priceBRL === 0 ? "GRATIS" : brl(cents(s.priceBRL))}
                        selected={s.serviceID === carrierID}
                        onSelect={() => setCarrierID(s.serviceID)}
                      />
                    ))}
                  </div>
                )}
              </div>
            </Section>
          )}

          <Section step={mode === "ship" ? "04" : "03"} title="Cupons">
            {accountStatus === "anon" ? (
              <p className="text-sm text-muted">
                <Link
                  href="/conta/entrar?next=/checkout"
                  onClick={() => trackLoginPrompt("checkout")}
                  className="text-brand-soft underline-offset-4 hover:underline"
                >
                  Entre com sua conta Collecta
                </Link>{" "}
                para usar cupons de desconto e de indicação.
              </p>
            ) : coupons === null ? (
              <p className="font-pixel text-[9px] text-brand-soft animate-blink">CARREGANDO CUPONS...</p>
            ) : (
              <>
                {coupons.length === 0 ? (
                  <p className="text-sm text-muted">
                    Você ainda não tem cupons de indicação liberados. Indique amigos no app e ganhe R$ 10 por indicação.
                  </p>
                ) : (
                  <>
                    <p className="mb-3 text-xs text-muted">
                      1 cupom a cada R$ 100 em itens
                      {applicable < coupons.length ? ` · ${applicable} aplicável${applicable === 1 ? "" : "is"} neste pedido` : ""}
                    </p>
                    <div className="grid gap-3">
                      {coupons.map((c) => {
                        const selected = selectedCoupons.includes(c.id);
                        const disabled = !selected && selectedCoupons.length >= applicable;
                        return (
                          <button
                            key={c.id}
                            type="button"
                            role="checkbox"
                            aria-checked={selected}
                            disabled={disabled}
                            onClick={() => toggleCoupon(c)}
                            className={`sticker flex w-full items-center justify-between gap-3 rounded-[10px] px-4 py-3 text-left transition-colors [--sh:3px] ${
                              selected ? "bg-brand text-white" : "bg-outline text-[#c9c9d1] hover:text-white"
                            } disabled:cursor-not-allowed disabled:opacity-50`}
                          >
                            <span className="min-w-0">
                              <span className="block truncate font-pixel text-[8px]">{c.code}</span>
                              <span className={`block text-xs ${selected ? "text-white/90" : "text-muted"}`}>
                                {selected ? "aplicado" : "toque para aplicar"}
                              </span>
                            </span>
                            <span className="shrink-0 font-pixel text-[11px]">−{brl(c.valueCents)}</span>
                          </button>
                        );
                      })}
                    </div>
                  </>
                )}
                <div className="mt-5 border-t-[3px] border-outline pt-5">
                  <h3 className="mb-3 font-pixel text-[9px] text-brand-soft">TEM UM CODIGO?</h3>
                  {promo ? (
                    <button
                      type="button"
                      onClick={removePromo}
                      className="sticker flex w-full items-center justify-between gap-3 rounded-[10px] bg-brand px-4 py-3 text-left text-white [--sh:3px]"
                    >
                      <span className="min-w-0">
                        <span className="block truncate font-pixel text-[8px]">{promo.code}</span>
                        <span className="block text-xs text-white/90">
                          {promo.percentBps / 100}% nos itens · toque para remover
                        </span>
                      </span>
                      <span className="shrink-0 font-pixel text-[11px]">−{brl(promoCents)}</span>
                    </button>
                  ) : (
                    <div className="flex items-start gap-3">
                      <Field
                        label="Código do cupom"
                        name="promoCode"
                        value={promoInput}
                        onChange={(v) => {
                          setPromoInput(v.toUpperCase());
                          setPromoError(null);
                        }}
                        error={promoError ?? undefined}
                        onEnter={() => void applyPromo()}
                        autoComplete="off"
                        maxLength={40}
                        placeholder="ex. BEMVINDO"
                        className="min-w-0 flex-1"
                      />
                      <button
                        type="button"
                        onClick={() => void applyPromo()}
                        disabled={promoChecking}
                        className="arcade-press sticker mt-6 shrink-0 rounded-[10px] bg-brand px-4 py-3 font-pixel text-[9px] text-white [--sh:3px] disabled:cursor-wait disabled:opacity-60"
                      >
                        {promoChecking ? "..." : "APLICAR"}
                      </button>
                    </div>
                  )}
                  {(selectedCoupons.length > 0 || promo) && (
                    <p className="mt-3 text-xs text-muted">
                      Os cupons não acumulam. Pelo WhatsApp o cupom é conferido manualmente na hora do pagamento.
                    </p>
                  )}
                </div>
              </>
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
              label={shippingLabel(mode, chosen, joinOrder)}
              value={chosen ? (shippingCents === 0 ? "Grátis" : brl(shippingCents)) : "—"}
            />
            {discountCents > 0 && (
              <Row
                label={`Cupom${discountCodes.length > 1 ? "s" : ""} · ${discountCodes.join(", ")}`}
                value={`−${brl(discountCents)}`}
              />
            )}
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
            type="submit"
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
      </form>
    </Container>
  );
}

function shippingLabel(mode: Mode, chosen: ShippingOption | null, joinOrder: string): string {
  if (mode === "pickup") {
    return "Retirada em mãos";
  }
  if (mode === "join") {
    return joinOrder ? `Junto com o pedido #${joinOrder}` : "Junto com outro pedido";
  }
  return chosen ? `Frete · ${chosen.name}` : "Frete";
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

function OptionChip({
  title,
  detail,
  price,
  selected,
  onSelect,
}: {
  title: string;
  detail: string;
  price: string;
  selected: boolean;
  onSelect: () => void;
}) {
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
        <span className="block truncate font-pixel text-[8px]">{pixelText(title)}</span>
        <span className={`block text-xs ${selected ? "text-white/90" : "text-muted"}`}>
          {detail}
        </span>
      </span>
      {price && <span className="shrink-0 font-pixel text-[11px]">{price}</span>}
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
