import type { SaleChannel, WriteOffReason } from "../api";
import { channelLabels, channels, writeOffLabels, writeOffs } from "../ligasales";
import { Select } from "./ui/select";

export type SaleExit = { kind: "sale"; channel: SaleChannel } | { kind: "writeOff"; reason: WriteOffReason };

export const DEFAULT_EXIT: SaleExit = { kind: "sale", channel: "balcao" };

function encode(exit: SaleExit): string {
  return exit.kind === "sale" ? `sale:${exit.channel}` : `writeOff:${exit.reason}`;
}

function decode(value: string): SaleExit | undefined {
  const channel = channels.find((c) => `sale:${c}` === value);
  if (channel) {
    return { kind: "sale", channel };
  }
  const reason = writeOffs.find((w) => `writeOff:${w}` === value);
  return reason ? { kind: "writeOff", reason } : undefined;
}

export function exitFields(exit: SaleExit): { channel?: SaleChannel; writeOff?: WriteOffReason } {
  return exit.kind === "sale" ? { channel: exit.channel } : { writeOff: exit.reason };
}

export default function SaleExitSelect({
  value,
  onChange,
  className,
}: {
  value: SaleExit;
  onChange: (next: SaleExit) => void;
  className?: string;
}) {
  return (
    <Select
      value={encode(value)}
      onChange={(e) => {
        const next = decode(e.target.value);
        if (next) {
          onChange(next);
        }
      }}
      className={className}
    >
      <optgroup label="Venda">
        {channels.map((c) => (
          <option key={c} value={`sale:${c}`}>
            {channelLabels[c]}
          </option>
        ))}
      </optgroup>
      <optgroup label="Baixa sem venda">
        {writeOffs.map((w) => (
          <option key={w} value={`writeOff:${w}`}>
            {writeOffLabels[w]}
          </option>
        ))}
      </optgroup>
    </Select>
  );
}
