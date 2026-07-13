import { useEffect, useState } from "react";
import { getTrackingSets } from "./api";

export function useSets(): string[] {
  const [sets, setSets] = useState<string[]>([]);
  useEffect(() => {
    let ok = true;
    getTrackingSets()
      .then((r) => {
        if (ok && r.sets.length > 0) {
          setSets(r.sets);
        }
      })
      .catch(() => {});
    return () => {
      ok = false;
    };
  }, []);
  return sets;
}
