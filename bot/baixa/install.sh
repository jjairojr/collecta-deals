#!/bin/sh
set -eu

here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
dest="/Users/Shared/collecta-bot/baixa"

mkdir -p "$dest/agente/.claude"
(cd "$repo" && go build -o "$dest/baixa-mcp.new" ./cmd/baixa-mcp)
mv "$dest/baixa-mcp.new" "$dest/baixa-mcp"
cp "$here/guard.sh" "$here/ferramentas-permitidas.txt" "$dest/"
cp "$here/settings.json" "$dest/agente/.claude/settings.json"
cp "$here/rotina.md" "$dest/agente/rotina.md"

chmod 755 "$dest" "$dest/agente" "$dest/agente/.claude" "$dest/baixa-mcp" "$dest/guard.sh"
chmod 644 "$dest/ferramentas-permitidas.txt" "$dest/agente/.claude/settings.json" "$dest/agente/rotina.md"

echo "Baixa instalada em $dest"
echo "Copie $dest/agente/.claude/settings.json para a pasta do agente no OpenBot (bot/baixa/SETUP.md, passo 3)."
