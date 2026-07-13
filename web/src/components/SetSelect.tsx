import { Select } from "./ui/select";

export const ALL_SETS = "ALL";

export default function SetSelect({
  sets,
  value,
  onChange,
  allowAll = false,
}: {
  sets: string[];
  value: string;
  onChange: (v: string) => void;
  allowAll?: boolean;
}) {
  if (sets.length <= 1) {
    return null;
  }
  return (
    <label className="flex flex-col gap-1 text-xs text-slate-400">
      Collection
      <Select value={value} onChange={(e) => onChange(e.target.value)} className="w-44">
        {allowAll && <option value={ALL_SETS}>All collections</option>}
        {sets.map((s) => (
          <option key={s} value={s}>
            {s}
          </option>
        ))}
      </Select>
    </label>
  );
}
