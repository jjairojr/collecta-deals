import Link from "next/link";
import CopyButton from "@/components/account/CopyButton";
import LogoutButton from "@/components/account/LogoutButton";
import TrackPageEvent from "@/components/analytics/TrackPageEvent";
import TrackedLink from "@/components/chrome/TrackedLink";
import Container from "@/components/ui/Container";
import type { Coupon, ReferralOverview, SessionUser } from "@/lib/collecta";
import { brl, pixelText } from "@/lib/format";
import { referralWhatsAppURL } from "@/lib/referral";

export default function AccountView({ user, referral }: { user: SessionUser; referral: ReferralOverview | null }) {
  const coupons = referral?.coupons ?? [];
  const unlocked = coupons.filter((c) => c.unlockedAt !== null && c.usedAt === null);
  const locked = coupons.filter((c) => c.unlockedAt === null);
  const used = coupons.filter((c) => c.usedAt !== null);
  const available = unlocked.reduce((sum, c) => sum + c.valueCents, 0);
  const lockedAsReferred = locked.some((c) => c.kind === "REFERRED");

  return (
    <Container narrow className="py-9">
      <TrackPageEvent
        event="view_account"
        unlocked={unlocked.length}
        locked={locked.length}
        used={used.length}
        referrals={referral?.referrals.length ?? 0}
      />
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <div className="font-pixel text-[9px] text-brand">PLAYER 1 · {pixelText(`@${user.username}`)}</div>
          <h1 className="mt-2 font-display text-4xl font-bold leading-none text-white sm:text-[52px]">
            {user.displayName || user.username}
          </h1>
          {user.email && <p className="mt-2 text-sm text-muted">{user.email}</p>}
        </div>
        <div className="flex items-center gap-3">
          <Link
            href="/conta/pedidos"
            className="arcade-press sticker rounded-[10px] bg-royal px-4 py-2.5 font-pixel text-[9px] text-white [--sh:4px]"
          >
            MEUS PEDIDOS
          </Link>
          <LogoutButton />
        </div>
      </div>

      <div className="mt-7 grid gap-6 lg:grid-cols-[1fr_1fr] lg:items-start">
        <section className="sticker sticker-5 rounded-[16px] border-brand bg-outline p-5 sm:p-7 [--sh:8px]">
          <div className="font-pixel text-[11px] text-brand-soft">INDIQUE E GANHE</div>
          {referral ? (
            <>
              <p className="mt-3 text-sm text-[#c9c9d1]">
                Cada amigo que criar conta com seu código e cadastrar {referral.activationCards} cartas
                libera {brl(referral.rewardCents)} para você e {brl(referral.rewardCents)} para ele, para
                usar aqui na loja. Até {referral.max} indicações.
              </p>
              <div className="mt-5 text-center font-pixel text-[22px] tracking-[6px] text-white sm:text-[28px]">
                {referral.code}
              </div>
              <p className="mt-2 truncate text-center text-xs text-muted">{referral.link}</p>
              <div className="mt-4 flex flex-wrap justify-center gap-3">
                <CopyButton text={referral.code} label="COPIAR CODIGO" share="copy_code" />
                <CopyButton text={referral.link} label="COPIAR LINK" share="copy_link" />
                <TrackedLink
                  href={referralWhatsAppURL(referral.code, referral.link)}
                  track={{ kind: "share_referral" }}
                  className="arcade-press sticker rounded-[10px] bg-brand px-4 py-2.5 font-pixel text-[9px] text-white [--sh:4px]"
                >
                  WHATSAPP
                </TrackedLink>
              </div>
              <p className="mt-5 text-center font-pixel text-[8px] text-faint">
                {pixelText(`${referral.referrals.length} DE ${referral.max} INDICACOES`)}
              </p>
            </>
          ) : (
            <p className="mt-3 text-sm text-muted">Não foi possível carregar seu programa de indicação agora.</p>
          )}
        </section>

        <section className="sticker rounded-[14px] bg-surface p-5 sm:p-6 [--sh:5px]">
          <div className="flex items-baseline justify-between gap-3">
            <h2 className="font-display text-2xl font-bold leading-none text-white">Cupons</h2>
            <span className="font-pixel text-[11px] text-brand">{brl(available)}</span>
          </div>
          <p className="mt-2 text-xs text-muted">1 cupom a cada R$ 100 em itens, no checkout do site.</p>

          <CouponGroup title="LIBERADOS" coupons={unlocked} tone="live" empty="Nenhum cupom liberado ainda." />
          {locked.length > 0 && (
            <CouponGroup
              title="TRAVADOS"
              coupons={locked}
              tone="locked"
              hint={
                lockedAsReferred && referral
                  ? `Cadastre ${referral.activationCards} cartas no app para liberar: ${Math.min(referral.catalogItems, referral.activationCards)}/${referral.activationCards}`
                  : "Libera quando o amigo indicado cadastrar as cartas no app."
              }
            />
          )}
          {used.length > 0 && <CouponGroup title="USADOS" coupons={used} tone="used" />}
          {unlocked.length > 0 && (
            <Link
              href="/checkout"
              className="arcade-press sticker mt-5 inline-block rounded-[10px] bg-brand px-5 py-3 font-pixel text-[10px] text-white [--sh:5px]"
            >
              USAR NO CHECKOUT
            </Link>
          )}
        </section>
      </div>
    </Container>
  );
}

function CouponGroup({
  title,
  coupons,
  tone,
  hint,
  empty,
}: {
  title: string;
  coupons: Coupon[];
  tone: "live" | "locked" | "used";
  hint?: string;
  empty?: string;
}) {
  return (
    <div className="mt-5">
      <h3 className="mb-2 font-pixel text-[8px] text-brand-soft">{title}</h3>
      {hint && <p className="mb-3 text-xs text-muted">{hint}</p>}
      {coupons.length === 0 ? (
        <p className="text-sm text-muted">{empty}</p>
      ) : (
        <ul className="grid gap-2">
          {coupons.map((c) => (
            <li
              key={c.id}
              className={`sticker flex items-center justify-between gap-3 rounded-[10px] px-4 py-3 [--sh:3px] ${
                tone === "live" ? "bg-brand text-white" : "bg-outline text-muted"
              }`}
            >
              <span className="min-w-0">
                <span className="block font-pixel text-[9px]">{c.code}</span>
                <span className={`block text-xs ${tone === "live" ? "text-white/90" : "text-muted"}`}>
                  {c.kind === "REFERRED" ? "boas-vindas" : c.referredUsername ? `indicação de @${c.referredUsername}` : "indicação"}
                  {tone === "used" ? " · usado" : ""}
                </span>
              </span>
              <span className="shrink-0 font-pixel text-[11px]">{brl(c.valueCents)}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
