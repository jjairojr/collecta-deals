#!/bin/sh
set -eu

here="$(cd "$(dirname "$0")" && pwd)"
repo="$(cd "$here/../.." && pwd)"
dest="/Users/Shared/collecta-bot/publicar"

mkdir -p "$dest/agente/.claude" "$dest/aprovados"
(cd "$repo" && go build -o "$dest/publicar-mcp.new" ./cmd/publicar-mcp)
mv "$dest/publicar-mcp.new" "$dest/publicar-mcp"
cp "$here/guard.sh" "$here/ferramentas-permitidas.txt" "$dest/"
cp "$here/settings.json" "$dest/agente/.claude/settings.json"
cp "$here/rotina.md" "$dest/agente/rotina.md"

chmod 755 "$dest" "$dest/agente" "$dest/agente/.claude" "$dest/aprovados" "$dest/publicar-mcp" "$dest/guard.sh"
chmod 644 "$dest/ferramentas-permitidas.txt" "$dest/agente/.claude/settings.json" "$dest/agente/rotina.md"

echo "Publicador instalado em $dest"
echo "Copie $dest/agente/.claude/settings.json para a pasta do agente no OpenBot (bot/publicar/SETUP.md, passo 3)."
