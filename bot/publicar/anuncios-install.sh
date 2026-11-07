#!/bin/sh
# Liga o com.collecta.anuncios no launchd: a cada 30 min sobe no Meta os testes aprovados, pausa os criativos que
# perdem e fecha os que terminaram. Só liga com o anuncios.env preenchido. Rode depois de make publicar.
set -eu

here="$(cd "$(dirname "$0")" && pwd)"
dest="/Users/Shared/collecta-bot/publicar"
env="$dest/anuncios.env"
label="com.collecta.anuncios"
plist="$HOME/Library/LaunchAgents/$label.plist"

if [ ! -f "$env" ]; then
  cp "$here/anuncios.env.exemplo" "$env"
  chmod 600 "$env"
  echo "Criei $env: preencha o token e os ids (bot/publicar/SETUP.md, Meta Ads) e rode de novo."
  exit 1
fi
chmod 600 "$env"
if ! grep -q '^META_TOKEN=..*' "$env"; then
  echo "META_TOKEN vazio em $env: preencha e rode de novo."
  exit 1
fi

cat > "$plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$label</string>
    <key>ProgramArguments</key>
    <array>
        <string>$dest/publicar-mcp</string>
        <string>anuncios</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PUBLICAR_ANUNCIOS_ENV</key>
        <string>$env</string>
    </dict>
    <key>WorkingDirectory</key>
    <string>$dest</string>
    <key>StartInterval</key>
    <integer>1800</integer>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>$dest/anuncios-launchd.log</string>
    <key>StandardErrorPath</key>
    <string>$dest/anuncios-launchd.log</string>
</dict>
</plist>
PLIST

launchctl bootout "gui/$(id -u)/$label" 2>/dev/null || true
launchctl bootstrap "gui/$(id -u)" "$plist"
echo "Testes de anúncio ligados: $label a cada 30 min (registro em $dest/anuncios.log)"
