#!/bin/sh
# Instala o vigia dos jobs em ~/.collecta-saude e liga o com.collecta.saude no launchd (a cada 15 min).
# Uso: scripts/saude/install.sh <tópico do ntfy>
set -eu

topic="${1:?uso: install.sh <tópico do ntfy>}"
here="$(cd "$(dirname "$0")" && pwd)"
dest="$HOME/.collecta-saude"
label="com.collecta.saude"
plist="$HOME/Library/LaunchAgents/$label.plist"

mkdir -p "$dest"
cp "$here/saude.sh" "$dest/saude.sh"
chmod 755 "$dest/saude.sh"

cat > "$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$label</string>
    <key>ProgramArguments</key>
    <array>
        <string>/bin/sh</string>
        <string>$dest/saude.sh</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>NTFY_TOPIC</key>
        <string>$topic</string>
        <key>PATH</key>
        <string>/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>
    <key>WorkingDirectory</key>
    <string>$dest</string>
    <key>StartInterval</key>
    <integer>900</integer>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardOutPath</key>
    <string>$dest/launchd.log</string>
    <key>StandardErrorPath</key>
    <string>$dest/launchd.log</string>
</dict>
</plist>
EOF

launchctl bootout "gui/$(id -u)/$label" 2>/dev/null || true
launchctl bootstrap "gui/$(id -u)" "$plist"
echo "Vigia instalado: $dest (log em $dest/saude.log)"
