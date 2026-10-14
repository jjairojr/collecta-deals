"use client";

import { useEffect } from "react";
import { trackViewAccount, trackViewOrders, trackViewStore } from "@/lib/analytics";

type PageEvent =
  | { event: "view_store" }
  | { event: "view_orders"; count: number }
  | { event: "view_account"; unlocked: number; locked: number; used: number; referrals: number };

export default function TrackPageEvent(props: PageEvent) {
  useEffect(() => {
    if (props.event === "view_store") {
      trackViewStore();
    } else if (props.event === "view_orders") {
      trackViewOrders(props.count);
    } else {
      trackViewAccount(props.unlocked, props.locked, props.used, props.referrals);
    }
  }, [props]);
  return null;
}
