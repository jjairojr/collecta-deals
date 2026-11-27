"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { Check, ChevronLeft } from "lucide-react";
import Container from "@/components/ui/Container";
import Field from "@/components/checkout/Field";
import UFSelect from "@/components/checkout/UFSelect";
import { ChoiceCard, ReviewBlock, StepCard, StepIndicator, type StepNumber } from "@/components/checkout/steps";
import { useAccount } from "@/lib/account";
import {
  trackAddPaymentInfo,
  trackAddShippingInfo,
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
import { brl } from "@/lib/format";
import { INTEREST_FREE, installmentsFor, shipsFromLabel, unitPriceFor, type PaymentMethod, SEM_JUROS } from "@/lib/payment";
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

async function quote(cep: string, lines: OrderLineInput[], paymentMethod: PaymentMethod): Promise<ShippingOption[]> {
  const res = await fetch("/api/frete", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ cep, lines, paymentMethod }),
  });
  const data = (await res.json()) as { services?: ShippingOption[]; error?: string };
  if (!res.ok) {
    throw new Error(data.error ?? "frete indisponível");
  }
  return data.services ?? [];
}

export default function CheckoutView({ joinOrder: initialJoin }: { joinOrder?: string }) {
  const { lines, reprice, cap } = useCart();
  const [payment, setPayment] = useState<PaymentMethod>("pix");
  const pixCents = lines.reduce((s, l) => s + unitPriceFor(l, "pix") * l.qty, 0);
  const cardCents = lines.reduce((s, l) => s + unitPriceFor(l, "card") * l.qty, 0);
  const itemsCents = payment === "card" ? cardCents : pixCents;
  const installments = installmentsFor(lines);
  const shipsFrom = lines.reduce((latest, l) => (l.shipsFrom && l.shipsFrom > latest ? l.shipsFrom : latest), "");
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
  const [mode, setMode] = useState<Mode>(initialJoin ? "join" : "ship");
  const [local, setLocal] = useState<ShippingOption[]>([]);
  const [carriers, setCarriers] = useState<ShippingOption[] | null>(null);
  const [quoting, setQuoting] = useState(false);
  const [quoteError, setQuoteError] = useState<string | null>(null);
  const [carrierID, setCarrierID] = useState<number | null>(null);
  const [joinOrder, setJoinOrder] = useState(initialJoin ?? "");
  const [joinError, setJoinError] = useState<string | null>(null);
  const [marketingOptIn, setMarketingOptIn] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const [step, setStep] = useState<StepNumber>(1);
  const [reached, setReached] = useState<StepNumber>(1);
  const stepHeading = useRef<HTMLHeadingElement>(null);
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
    quote("", payloadLines, payment)
      .then((services) => {
        if (!cancelled) {
          const found = services.filter((s) => s.serviceID < 0);
          setLocal(found);
          if (!found.some((s) => isJoin(s.serviceID))) {
            setMode((m) => (m === "join" ? "ship" : m));
          }
        }
      })
      .catch(() => {
        if (!cancelled) {
          setLocal([]);
          setMode((m) => (m === "join" ? "ship" : m));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [payloadLines, payment]);

  useEffect(() => {
    if (mode !== "ship" || cepDigits.length !== 8 || payloadLines.length === 0) {
      setCarriers(null);
      setCarrierID(null);
      return;
    }
    let cancelled = false;
    setQuoting(true);
    setQuoteError(null);
    quote(cepDigits, payloadLines, payment)
      .then((services) => {
        if (cancelled) {
          return;
        }
        const found = services.filter((s) => s.serviceID >= 0);
        setCarriers(found);
        // Keep the customer's pick across re-quotes; never pick for them, unless
        // there is only one way to ship.
        setCarrierID((prev) =>
          found.some((s) => s.serviceID === prev) ? prev : found.length === 1 ? found[0].serviceID : null,
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
  }, [mode, cepDigits, payloadLines, payment]);

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
  // What the customer pays with each method, with the shipping chosen now (the
  // quote follows the method, so it can still change after switching).
  const totalFor = (m: PaymentMethod) => {
    const items = m === "card" ? cardCents : pixCents;
    const off = promo ? promoDiscountCents(items, promo.percentBps) : activeCoupons.reduce((sum, c) => sum + c.valueCents, 0);
    return Math.max(items + shippingCents - off, 0);
  };
  const scrollToStep = useCallback(() => {
    window.scrollTo({ top: 0, behavior: "smooth" });
    window.setTimeout(() => stepHeading.current?.focus({ preventScroll: true }), 250);
  }, []);

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
          marketingOptIn,
          paymentMethod: payment,
        }),
      });
      const data = (await res.json()) as CreateResponse & CheckoutError;
      if (res.ok && data.initPoint) {
        trackGenerateLead(totalCents, "mercadopago");
        window.location.assign(data.initPoint);
        return;
      }
      if (data.error === "price_changed" && data.lines) {
        const prices: Record<string, { price: number; cardPrice: number }> = {};
        for (const l of data.lines) {
          if (typeof l.unitCents === "number") {
            prices[l.sku] = { price: l.unitCents, cardPrice: l.cardCents ?? l.unitCents };
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
        setCarrierID(null);
        setStep(2);
        scrollToStep();
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

  const goTo = (n: StepNumber) => {
    setNotice(null);
    setStep(n);
    scrollToStep();
  };

  // Validates the current step and moves on; the last step pays.
  const next = () => {
    if (step === 1) {
      const be = validateBuyer(buyer);
      setBuyerErrors(be);
      if (Object.keys(be).length > 0) {
        setNotice("Confira os campos destacados.");
        return;
      }
    } else if (step === 2) {
      const ae = mode === "ship" ? validateAddress(address) : {};
      setAddressErrors(ae);
      if (Object.keys(ae).length > 0) {
        setNotice("Confira o endereço.");
        return;
      }
      if (mode === "join" && !/^\d+$/.test(joinOrder.trim())) {
        setJoinError("Número do pedido");
        setNotice("Informe o número do pedido pago que vai junto.");
        return;
      }
      if (!chosen) {
        setNotice(mode === "ship" ? "Escolha uma opção de frete." : "Escolha como quer receber.");
        return;
      }
      trackAddShippingInfo(lines, totalCents, shippingLabel(mode, chosen, joinOrder));
    } else if (step === 3) {
      trackAddPaymentInfo(lines, totalCents, payment === "pix" ? "pix" : "credit_card");
    } else {
      void submit();
      return;
    }
    const n = (step + 1) as StepNumber;
    setNotice(null);
    setStep(n);
    setReached((r) => (n > r ? n : r));
    scrollToStep();
  };

  const actionLabel =
    step === 4 ? (submitting ? "ABRINDO PAGAMENTO..." : "PAGAR COM MERCADO PAGO") : "CONTINUAR";
  const actionDisabled = submitting || (step >= 2 && quoting);
  const methodNote =
    payment === "pix" ? "no Pix" : `no cartão, em até ${installments}x${SEM_JUROS}`;
  const deliveryDays = chosen && chosen.serviceID >= 0 ? chosen.deliveryDays : 0;

  return (
    <Container narrow className="pb-32 pt-9 lg:pb-9">
      <h1 className="font-display text-4xl font-bold leading-none text-white sm:text-[52px]">
        Checkout
      </h1>
      <div className="mt-3 font-pixel text-[9px] text-brand">PLAYER 1 · 4 PASSOS PARA PAGAR</div>

      <div className="mt-6">
        <StepIndicator current={step} reached={reached} onGo={goTo} />
      </div>

      <form
        className="mt-6 grid gap-7 lg:grid-cols-[1.4fr_.6fr] lg:items-start"
        autoComplete="on"
        noValidate
        onSubmit={(e) => {
          e.preventDefault();
          next();
        }}
      >
        <div className="min-w-0 space-y-5">
          {step === 1 && (
            <StepCard step={1} title="Seus dados" headingRef={stepHeading}>
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
                <button
                  type="button"
                  role="checkbox"
                  aria-checked={marketingOptIn}
                  onClick={() => setMarketingOptIn((v) => !v)}
                  className="flex min-h-[44px] items-center gap-3 text-left text-sm text-[#c9c9d1] transition-colors hover:text-white sm:col-span-2"
                >
                  <span
                    aria-hidden
                    className={`sticker flex h-6 w-6 shrink-0 items-center justify-center rounded-[6px] [--sh:2px] ${
                      marketingOptIn ? "bg-brand text-white" : "bg-outline"
                    }`}
                  >
                    {marketingOptIn && <Check size={14} strokeWidth={3} />}
                  </span>
                  Quero receber ofertas da Collecta pelo WhatsApp
                </button>
              </div>
            </StepCard>
          )}

          {step === 2 && (
            <StepCard step={2} title="Entrega" headingRef={stepHeading}>
              <h3 className="mb-3 font-pixel text-[9px] text-brand-soft">COMO RECEBER</h3>
              <div className="grid gap-3" role="radiogroup" aria-label="Como receber">
                {modes.map((m) => (
                  <ChoiceCard
                    key={m.key}
                    title={m.title}
                    lines={[m.detail]}
                    price={m.price || undefined}
                    selected={mode === m.key}
                    onSelect={() => setMode(m.key)}
                  />
                ))}
              </div>

              {mode === "pickup" && (
                <p className="mt-4 text-sm text-muted">
                  Depois do pagamento a gente combina dia e local da retirada pelo WhatsApp. Sem endereço, sem frete.
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
                    Só vale para pedido já pago e ainda não enviado, no mesmo CPF ou e-mail. Vai para o mesmo endereço,
                    numa encomenda só.
                  </p>
                </div>
              )}

              {mode === "ship" && (
                <>
                  <div className="mt-6 grid gap-4 sm:grid-cols-6">
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
                    <p className="self-end pb-3 text-xs text-muted sm:col-span-4">
                      Com o CEP a gente calcula o frete e preenche o endereço.
                    </p>
                  </div>

                  <div className="mt-5">
                    <h3 className="mb-3 font-pixel text-[9px] text-brand-soft">ESCOLHA O FRETE</h3>
                    {cepDigits.length !== 8 ? (
                      <p className="text-sm text-muted">Digite o CEP para ver as opções, o preço e o prazo de cada uma.</p>
                    ) : quoting ? (
                      <p className="font-pixel text-[9px] text-brand-soft animate-blink">CALCULANDO FRETE...</p>
                    ) : quoteError ? (
                      <p className="text-sm text-brand">
                        Não conseguimos cotar o frete agora ({quoteError}). Tente de novo ou feche pelo WhatsApp.
                      </p>
                    ) : carriers && carriers.length === 0 ? (
                      <p className="text-sm text-brand">
                        Nenhuma transportadora atende esse CEP com esse pacote. Fale com a gente no WhatsApp.
                      </p>
                    ) : (
                      <div className="grid gap-3" role="radiogroup" aria-label="Opções de frete">
                        {carriers?.map((s) => {
                          const free = s.priceBRL === 0;
                          return (
                            <ChoiceCard
                              key={s.serviceID}
                              title={`${s.company} · ${s.name}`}
                              lines={[
                                `chega em até ${s.deliveryDays} ${s.deliveryDays === 1 ? "dia útil" : "dias úteis"} depois da postagem`,
                                ...(free ? ["frete grátis nesta compra"] : []),
                              ]}
                              price={free ? "GRATIS" : brl(cents(s.priceBRL))}
                              was={free && s.quotedBRL ? brl(cents(s.quotedBRL)) : undefined}
                              selected={s.serviceID === carrierID}
                              onSelect={() => {
                                setCarrierID(s.serviceID);
                                setNotice(null);
                              }}
                            />
                          );
                        })}
                      </div>
                    )}
                  </div>

                  <div className="mt-6 border-t-[3px] border-outline pt-5">
                    <h3 className="mb-3 font-pixel text-[9px] text-brand-soft">ENDERECO DE ENTREGA</h3>
                    <div className="grid gap-4 sm:grid-cols-6">
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
                        className="sm:col-span-2"
                      />
                      <Field
                        label="Bairro"
                        name="district"
                        value={address.district}
                        onChange={(v) => setAddressField("district", v)}
                        error={addressErrors.district}
                        autoComplete="address-level3"
                        className="sm:col-span-4"
                      />
                      <Field
                        label="Cidade"
                        name="city"
                        value={address.city}
                        onChange={(v) => setAddressField("city", v)}
                        error={addressErrors.city}
                        autoComplete="address-level2"
                        className="sm:col-span-4"
                      />
                      <div className="sm:col-span-2">
                        <UFSelect
                          value={address.state}
                          onChange={(v) => setAddressField("state", v)}
                          error={addressErrors.state}
                        />
                      </div>
                    </div>
                  </div>
                </>
              )}
            </StepCard>
          )}

          {step === 3 && (
            <StepCard step={3} title="Pagamento" headingRef={stepHeading}>
              <div className="grid gap-3" role="radiogroup" aria-label="Forma de pagamento">
                <ChoiceCard
                  title="Pix"
                  lines={["aprovação na hora", `itens ${brl(pixCents)}${chosen ? ` + frete ${shippingCents === 0 ? "grátis" : brl(shippingCents)}` : ""}`]}
                  price={brl(totalFor("pix"))}
                  selected={payment === "pix"}
                  onSelect={() => setPayment("pix")}
                />
                <ChoiceCard
                  title="Cartão de crédito"
                  lines={[
                    INTEREST_FREE
                      ? `em até ${installments}x de ${brl(Math.round(totalFor("card") / installments))} sem juros`
                      : `em até ${installments}x · juros, se houver, na tela do Mercado Pago`,
                    `itens ${brl(cardCents)}${chosen ? ` + frete ${shippingCents === 0 ? "grátis" : brl(shippingCents)}` : ""}`,
                  ]}
                  price={brl(totalFor("card"))}
                  selected={payment === "card"}
                  onSelect={() => setPayment("card")}
                />
              </div>
              <p className="mt-4 text-sm text-muted">
                {cardCents !== pixCents ? "O preço dos itens muda conforme a forma de pagamento. " : ""}
                Você escolhe o número de parcelas na tela do Mercado Pago, depois de revisar o pedido.
              </p>

              <div className="mt-6 border-t-[3px] border-outline pt-5">
                <h3 className="mb-3 font-pixel text-[9px] text-brand-soft">CUPONS</h3>
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
                                className={`sticker flex min-h-[52px] w-full items-center justify-between gap-3 rounded-[10px] px-4 py-3 text-left transition-colors [--sh:3px] ${
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
                    <div className="mt-5">
                      {promo ? (
                        <button
                          type="button"
                          onClick={removePromo}
                          className="sticker flex min-h-[52px] w-full items-center justify-between gap-3 rounded-[10px] bg-brand px-4 py-3 text-left text-white [--sh:3px]"
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
                            label="Tem um código?"
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
                            className="arcade-press sticker mt-6 min-h-[48px] shrink-0 rounded-[10px] bg-brand px-4 py-3 font-pixel text-[9px] text-white [--sh:3px] disabled:cursor-wait disabled:opacity-60"
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
              </div>
            </StepCard>
          )}

          {step === 4 && (
            <StepCard step={4} title="Revisão" headingRef={stepHeading}>
              <div className="grid gap-3">
                <ReviewBlock title="Seus dados" onEdit={() => goTo(1)}>
                  <p>{buyer.name}</p>
                  <p className="text-muted">
                    {buyer.email} · {buyer.phone}
                  </p>
                </ReviewBlock>
                <ReviewBlock title="Entrega" onEdit={() => goTo(2)}>
                  <p>
                    {shippingLabel(mode, chosen, joinOrder)} ·{" "}
                    {chosen ? (shippingCents === 0 ? "grátis" : brl(shippingCents)) : "—"}
                  </p>
                  {mode === "ship" && (
                    <p className="text-muted">
                      {address.street}, {address.number}
                      {address.complement ? ` · ${address.complement}` : ""} · {address.district} · {address.city}/
                      {address.state} · CEP {address.cep}
                    </p>
                  )}
                  {deliveryDays > 0 && (
                    <p className="text-muted">
                      Chega em até {deliveryDays} {deliveryDays === 1 ? "dia útil" : "dias úteis"} depois da postagem.
                    </p>
                  )}
                </ReviewBlock>
                <ReviewBlock title="Pagamento" onEdit={() => goTo(3)}>
                  <p>{payment === "pix" ? "Pix · aprovação na hora" : `Cartão de crédito · em até ${installments}x${SEM_JUROS}`}</p>
                  <p className="text-muted">
                    {payment === "pix"
                      ? "O QR code do Pix aparece na tela do Mercado Pago."
                      : "Você escolhe as parcelas na tela do Mercado Pago, que mostra o valor de cada uma."}
                  </p>
                </ReviewBlock>
              </div>
            </StepCard>
          )}
        </div>

        <div
          className={`sticker sticker-5 min-w-0 rounded-[16px] border-brand bg-outline p-5 sm:p-[26px] lg:sticky lg:top-24 ${
            step === 4 ? "" : "hidden lg:block"
          }`}
          style={{ ["--sh" as string]: "8px" }}
        >
          <div className="font-pixel text-[11px] text-brand-soft">RESUMO</div>

          <ul className="mt-4 space-y-2 text-sm">
            {lines.map((l) => (
              <li key={`${l.sku}|${l.meta}`} className="flex items-start justify-between gap-3">
                <span className="min-w-0">
                  <span className="block truncate text-white">
                    {l.qty}× {l.name}
                  </span>
                  <span className="block truncate text-xs text-muted">{l.meta}</span>
                  {l.shipsFrom && (
                    <span className="block text-xs font-bold text-brand">
                      Pré-venda · envio a partir de {shipsFromLabel(l.shipsFrom)}
                    </span>
                  )}
                </span>
                <span className="shrink-0 font-pixel text-[10px] text-white">
                  {brl(unitPriceFor(l, payment) * l.qty)}
                </span>
              </li>
            ))}
          </ul>

          <dl className="mt-5 space-y-3 border-t-[3px] border-brand pt-5 text-[15px]">
            <Row label="Itens" value={brl(itemsCents)} />
            <Row
              label={shippingLabel(mode, chosen, joinOrder)}
              value={chosen ? (shippingCents === 0 ? "Grátis" : brl(shippingCents)) : "escolha na etapa 2"}
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
            <span className="font-pixel text-[10px] text-white">{chosen ? "TOTAL" : "TOTAL SEM FRETE"}</span>
            <span className="font-pixel text-[18px] text-brand sm:text-[22px]">{brl(totalCents)}</span>
          </div>
          <p className="mt-2 text-xs text-muted">{methodNote}</p>
          {shipsFrom && (
            <p className="mt-3 rounded-[8px] border-[3px] border-brand px-3 py-2 text-xs text-white">
              Este pedido tem item de pré-venda: o envio ou a retirada acontece a partir de{" "}
              {shipsFromLabel(shipsFrom)}, junto com o restante do pedido.
            </p>
          )}

          {notice && (
            <p
              role="alert"
              className="sticker mt-4 hidden rounded-[10px] bg-brand-soft px-3 py-2 text-xs font-semibold text-outline [--sh:3px] lg:block"
            >
              {notice}
            </p>
          )}

          <button
            type="submit"
            disabled={actionDisabled}
            className="mt-5 hidden w-full rounded-[10px] border-4 border-white bg-brand px-6 py-3.5 font-pixel text-[11px] text-white transition-transform hover:-translate-y-0.5 disabled:cursor-wait disabled:opacity-60 lg:block"
            style={{ boxShadow: "0 0 0 4px #0b0b0c" }}
          >
            {actionLabel}
          </button>
          {step > 1 && (
            <button
              type="button"
              onClick={() => goTo((step - 1) as StepNumber)}
              className="mt-3 hidden min-h-[44px] w-full text-center text-sm text-muted underline-offset-4 hover:text-brand-soft hover:underline lg:block"
            >
              Voltar
            </button>
          )}
          {step === 4 && (
            <>
              <button
                type="button"
                onClick={whatsapp}
                className="mt-4 block min-h-[44px] w-full rounded px-2 py-2 text-center text-xs text-muted underline-offset-4 hover:text-brand-soft hover:underline"
              >
                ou feche o pedido pelo WhatsApp
              </button>
              <p className="mt-3 text-center font-pixel text-[8px] leading-relaxed text-faint">
                VOCE SERA REDIRECIONADO AO MERCADO PAGO
                <br />
                ESTOQUE RESERVADO POR 2 HORAS
              </p>
            </>
          )}
        </div>

        {/* Phone: the total and the next action always visible at the bottom. */}
        <div className="fixed inset-x-0 bottom-0 z-40 border-t-4 border-brand bg-outline px-4 pb-[max(12px,env(safe-area-inset-bottom))] pt-3 lg:hidden">
          {notice && (
            <p role="alert" className="mb-2 rounded-[8px] bg-brand-soft px-3 py-2 text-xs font-semibold text-outline">
              {notice}
            </p>
          )}
          <div className="flex items-center gap-3">
            {step > 1 && (
              <button
                type="button"
                onClick={() => goTo((step - 1) as StepNumber)}
                className="flex h-12 w-12 shrink-0 items-center justify-center rounded-[10px] border-[3px] border-[#3a3a42] text-[#c9c9d1]"
                aria-label="Voltar para a etapa anterior"
              >
                <ChevronLeft size={22} strokeWidth={3} aria-hidden />
              </button>
            )}
            <div className="min-w-0 flex-1">
              <div className="font-pixel text-[8px] text-muted">{chosen ? "TOTAL" : "SEM FRETE"}</div>
              <div className="whitespace-nowrap font-pixel text-[13px] text-brand">{brl(totalCents)}</div>
              <div className="truncate text-[11px] text-muted">{methodNote}</div>
            </div>
            <button
              type="submit"
              disabled={actionDisabled}
              className="min-h-[48px] shrink-0 rounded-[10px] bg-brand px-4 font-pixel text-[10px] text-white disabled:cursor-wait disabled:opacity-60"
            >
              {step === 4 ? (submitting ? "ABRINDO..." : "PAGAR") : "CONTINUAR"}
            </button>
          </div>
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

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <dt className="truncate text-[#c9c9d1]">{label}</dt>
      <dd className="shrink-0 text-white">{value}</dd>
    </div>
  );
}
