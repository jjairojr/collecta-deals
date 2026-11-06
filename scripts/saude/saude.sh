#!/bin/sh
# Vigia dos jobs com.collecta.* do launchd. Roda fora de ~/Documents (o macOS bloqueia o launchd lá).
# Avisa no ntfy só quando um job muda de estado: ok -> parado ou parado -> ok.
set -u

dir="$HOME/.collecta-saude"
state="$dir/estado"
log="$dir/saude.log"
topic="${NTFY_TOPIC:?defina NTFY_TOPIC}"
self="com.collecta.saude"

mkdir -p "$dir"
touch "$state"

now="$(date '+%Y-%m-%d %H:%M:%S')"
current="$(launchctl list | awk -v self="$self" '$3 ~ /^com\.collecta\./ && $3 != self {
  if ($1 != "-" || $2 == "0") print $3, "ok", $2; else print $3, "parado", $2
}')"

printf '%s\n' "$current" | while read -r label status code; do
  [ -z "$label" ] && continue
  before="$(awk -v l="$label" '$1 == l {print $2}' "$state")"
  if [ "$before" != "$status" ] && { [ -n "$before" ] || [ "$status" = "parado" ]; }; then
    printf '%s %s: %s -> %s (saída %s)\n' "$now" "$label" "${before:-novo}" "$status" "$code" >> "$log"
    printf '%s: %s (saída %s)\n' "$label" "$status" "$code"
  fi
done > "$dir/mudancas"

printf '%s\n' "$current" | awk 'NF {print $1, $2}' > "$state.tmp" && mv "$state.tmp" "$state"

if [ -s "$dir/mudancas" ]; then
  priority=default
  grep -q ' parado ' "$dir/mudancas" && priority=high
  if ! curl -fsS -m 20 -H "Title: Jobs da Collecta" -H "Priority: $priority" -H "Tags: gear" \
    --data-binary @"$dir/mudancas" "https://ntfy.sh/$topic" > /dev/null; then
    printf '%s ntfy falhou, aviso fica para a próxima rodada\n' "$now" >> "$log"
    : > "$state"
  fi
fi
rm -f "$dir/mudancas"
