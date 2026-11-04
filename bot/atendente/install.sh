#!/bin/sh
set -eu

here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
dest="/Users/Shared/collecta-bot"

mkdir -p "$dest/agente/.claude"
(cd "$repo" && go build -o "$dest/atendente-mcp.new" ./cmd/atendente-mcp)
mv "$dest/atendente-mcp.new" "$dest/atendente-mcp"
(cd "$repo" && go build -o "$dest/atendente-vigia.new" ./cmd/atendente-vigia)
mv "$dest/atendente-vigia.new" "$dest/atendente-vigia"
cp "$here/guard.sh" "$here/ferramentas-permitidas.txt" "$dest/"
cp "$here/settings.json" "$dest/agente/.claude/settings.json"
cp "$here/rotina.md" "$dest/agente/rotina.md"
if [ ! -f "$dest/novidades.md" ]; then
  cp "$here/novidades.md" "$dest/novidades.md"
  chmod 644 "$dest/novidades.md"
fi

chmod 755 "$dest" "$dest/agente" "$dest/agente/.claude" "$dest/atendente-mcp" "$dest/atendente-vigia" "$dest/guard.sh"
chmod 644 "$dest/ferramentas-permitidas.txt" "$dest/agente/.claude/settings.json" "$dest/agente/rotina.md"

data="$HOME/.collecta-atendente"
env="$data/vigia.env"
plist="$HOME/Library/LaunchAgents/com.collecta.atendente-vigia.plist"
mkdir -p "$data" "$HOME/Library/LaunchAgents"
chmod 700 "$data"
if [ ! -f "$env" ]; then
  cp "$here/vigia.env.example" "$env"
fi
chmod 600 "$env"
sed "s|__HOME__|$HOME|g" "$here/vigia.plist" > "$plist"
if grep -Eq '^POSTSALE_AGENT_TOKEN=.+' "$env"; then
  launchctl bootout "gui/$(id -u)/com.collecta.atendente-vigia" 2>/dev/null || true
  launchctl bootstrap "gui/$(id -u)" "$plist"
  echo "Vigia rodando (log em $data/vigia.log)"
else
  echo "Vigia instalado mas parado: preencha POSTSALE_AGENT_TOKEN em $env e rode make atendente de novo"
fi

echo "Atendente instalado em $dest"
echo "Copie $dest/agente/.claude/settings.json para a pasta do agente no OpenBot (SETUP.md, passo 5)."
