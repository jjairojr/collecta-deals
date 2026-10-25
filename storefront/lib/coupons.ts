export const REFERRAL_MIN_ORDER_CENTS = 10000;

export interface CheckoutCoupon {
  id: string;
  code: string;
  valueCents: number;
  minOrderCents: number;
}

export function couponsApplicable(
  available: number,
  totalCents: number,
  minOrderCents = REFERRAL_MIN_ORDER_CENTS,
): number {
  if (available <= 0 || minOrderCents <= 0) {
    return 0;
  }
  return Math.min(available, Math.floor(totalCents / minOrderCents));
}

export interface PromoCoupon {
  code: string;
  percentBps: number;
}

export function promoDiscountCents(itemsCents: number, percentBps: number): number {
  return Math.floor((itemsCents * percentBps) / 10000);
}
