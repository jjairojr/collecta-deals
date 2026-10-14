import Link from "next/link";
import { redirect } from "next/navigation";
import Container from "@/components/ui/Container";
import { checkoutFetch } from "@/lib/backend";
import { type OrderStatus, type PublicOrder, cents } from "@/lib/checkout";
import { brl, pixelText } from "@/lib/format";
import { sessionUser } from "@/lib/session";

export const metadata = {
  title: "Meus pedidos",
  robots: { index: false },
};

const STATUS_LABEL: Record<OrderStatus, string> = {
  pending: "AGUARDANDO PAGAMENTO",
  paid: "PAGO",
  failed: "PAGAMENTO RECUSADO",
  expired: "EXPIRADO",
  cancelled: "CANCELADO",
  oversold: "EM CONFERENCIA",
  shipped: "ENVIADO",
  refunded: "ESTORNADO",
};

async function loadOrders(userID: string): Promise<PublicOrder[] | null> {
  const res = await checkoutFetch(`/api/orders?collectaUserId=${encodeURIComponent(userID)}`);
  if (!res.ok) {
    return null;
  }
  return (await res.json()) as PublicOrder[];
}

export default async function PedidosPage() {
  const session = await sessionUser();
  if (!session) {
    redirect("/conta/entrar?next=/conta/pedidos");
  }
  const orders = await loadOrders(session.user.id);
  return (
    <Container narrow className="py-9">
      <div className="font-pixel text-[9px] text-brand">PLAYER 1 · HISTORICO</div>
      <h1 className="mt-2 font-display text-4xl font-bold leading-none text-white sm:text-[52px]">Meus pedidos</h1>
      <Link href="/conta" className="mt-3 inline-block text-xs text-muted underline-offset-4 hover:text-brand-soft hover:underline">
        voltar para a conta
      </Link>

      {orders === null ? (
        <p className="mt-7 text-sm text-muted">Não foi possível carregar seus pedidos agora.</p>
      ) : orders.length === 0 ? (
        <p className="mt-7 text-sm text-muted">Você ainda não fez pedidos com essa conta.</p>
      ) : (
        <ul className="mt-7 grid gap-4">
          {orders.map((o) => (
            <li key={o.id} className="sticker rounded-[14px] bg-surface p-5 [--sh:5px]">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <div className="font-pixel text-[9px] text-brand-soft">
                    {pixelText(`PEDIDO #${o.number} · ${STATUS_LABEL[o.status]}`)}
                  </div>
                  <div className="mt-1 text-xs text-muted">
                    {new Date(o.createdAt).toLocaleDateString("pt-BR")} · {o.lines.reduce((n, l) => n + l.qty, 0)} item(ns)
                  </div>
                </div>
                <div className="text-right">
                  <div className="font-pixel text-[14px] text-brand">{brl(cents(o.totals.totalBRL))}</div>
                  {o.totals.discountBRL > 0 && (
                    <div className="text-xs text-muted">cupom −{brl(cents(o.totals.discountBRL))}</div>
                  )}
                </div>
              </div>
              <ul className="mt-3 space-y-2 text-sm text-[#c9c9d1]">
                {o.lines.map((l, i) => (
                  <li key={`${l.name}-${i}`} className="flex items-center gap-3">
                    {l.imageURL ? (
                      <img
                        src={l.imageURL}
                        alt={l.name}
                        width={64}
                        height={90}
                        loading="lazy"
                        className="h-[90px] w-16 shrink-0 rounded-[6px] border-[3px] border-outline object-cover"
                      />
                    ) : (
                      <span className="art-stripes-90 block h-[90px] w-16 shrink-0 rounded-[6px] border-[3px] border-outline bg-royal" aria-hidden />
                    )}
                    <span className="truncate">
                      {l.qty}× {l.name}
                      {l.meta ? <span className="text-muted"> · {l.meta}</span> : null}
                    </span>
                  </li>
                ))}
              </ul>
              <Link
                href={`/pedido/${o.id}`}
                className="arcade-press sticker mt-4 inline-block rounded-[10px] bg-outline px-4 py-2 font-pixel text-[8px] text-white [--sh:3px]"
              >
                VER PEDIDO
              </Link>
            </li>
          ))}
        </ul>
      )}
    </Container>
  );
}
