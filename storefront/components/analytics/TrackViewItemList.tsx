"use client";

import { useEffect } from "react";
import { catalogItem, trackViewItemList } from "@/lib/analytics";
import type { Accessory, Sealed, Single } from "@/lib/types";

export default function TrackViewItemList({
  items,
  listId,
  listName,
}: {
  items: (Single | Sealed | Accessory)[];
  listId: string;
  listName: string;
}) {
  useEffect(() => {
    if (items.length > 0) {
      trackViewItemList(items.map(catalogItem), listId, listName);
    }
  }, [items, listId, listName]);

  return null;
}
