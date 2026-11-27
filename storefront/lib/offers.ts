import { kitStock } from "@/lib/cart-lines";
import type { Accessory, KitOffer, Sealed } from "@/lib/types";

// Offers shown around the cart. They only pick real, in-stock catalog items: when
// nothing qualifies the offer is simply not shown.

// Accessory offered in the "added to cart" toast after a single or a slab.
const CARD_ACCESSORY = /sleeve|toploader/i;

export function pickCardAccessory(accessories: Accessory[]): Accessory | undefined {
  return accessories
    .filter((a) => a.qty > 0 && a.price > 0 && CARD_ACCESSORY.test(a.name))
    .sort((a, b) => a.price - b.price)[0];
}

// Booster kit suggested in the cart: the 30th anniversary booster (not the booster
// box or a display), cheapest kit that still has stock.
const BOOSTER_30 = /30 anos/i;
const BOOSTER = /booster/i;
const NOT_BOOSTER_PACK = /\bbox\b|display/i;

export interface BoosterKitOffer {
  item: Sealed;
  kit: KitOffer;
}

export function pickBoosterKit(sealed: Sealed[]): BoosterKitOffer | undefined {
  let best: BoosterKitOffer | undefined;
  for (const item of sealed) {
    if (!BOOSTER_30.test(item.name) || !BOOSTER.test(item.name) || NOT_BOOSTER_PACK.test(item.name)) {
      continue;
    }
    for (const kit of item.kits ?? []) {
      if (kitStock(item.qty, kit.units) < 1) {
        continue;
      }
      if (!best || kit.price < best.kit.price) {
        best = { item, kit };
      }
    }
  }
  return best;
}
