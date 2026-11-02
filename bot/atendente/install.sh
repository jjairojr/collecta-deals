#!/bin/sh
set -eu

here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
dest="/Users/Shared/collecta-bot"

mkdir -p "$dest/agente/.claude"
(cd "$repo" && go build -o "$dest/atendente-mcp.new" ./cmd/atendente-mcp)
mv "$dest/atendente-mcp.new" "$dest/atendente-mcp"
cp "$here/guard.sh" "$here/ferramentas-permitidas.txt" "$dest/"
cp "$here/settings.json" "$dest/agente/.claude/settings.json"
cp "$here/rotina.md" "$dest/agente/rotina.md"
if [ ! -f "$dest/novidades.md" ]; then
  cp "$here/novidades.md" "$dest/novidades.md"
  chmod 644 "$dest/novidades.md"
fi

chmod 755 "$dest" "$dest/agente" "$dest/agente/.claude" "$dest/atendente-mcp" "$dest/guard.sh"
chmod 644 "$dest/ferramentas-permitidas.txt" "$dest/agente/.claude/settings.json" "$dest/agente/rotina.md"

echo "Atendente instalado em $dest"
echo "Copie $dest/agente/.claude/settings.json para a pasta do agente no OpenBot (SETUP.md, passo 5)."
