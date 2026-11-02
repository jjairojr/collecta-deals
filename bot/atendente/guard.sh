#!/bin/sh
here="$(cd "$(dirname "$0")" && pwd)"
allowlist="$here/ferramentas-permitidas.txt"
logdir="${ATENDENTE_DADOS:-$HOME/.collecta-atendente}"
input="$(cat)"

field() {
  printf '%s' "$input" | /usr/bin/plutil -extract "$1" raw -o - - 2>/dev/null
}

block() {
  mkdir -p "$logdir"
  printf '%s bloqueado: %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$1" >> "$logdir/bloqueios.log"
  echo "Bloqueado no atendente: $1. Use só o navegador no WhatsApp Web e as ferramentas da collecta-loja." >&2
  exit 2
}

tool="$(field tool_name)"

if [ "$tool" = "mcp__openbot__tool_call" ]; then
  target="$(field tool_input.name)"
  case "$target" in
    openbot_browser.upload_files | openbot_browser.run_script | openbot_browser.evaluate | openbot_browser.fill_fields) block "$target" ;;
    openbot_browser.*login* | openbot_browser.*secret* | openbot_browser.*download* | openbot_browser.*vault*) block "$target" ;;
    openbot_browser.*) ;;
    *) block "tool_call ${target:-sem nome}" ;;
  esac
  url="$(field tool_input.arguments.url)"
  if [ -n "$url" ]; then
    case "$url" in
      https://web.whatsapp.com | https://web.whatsapp.com/*) ;;
      *) block "$target fora do WhatsApp Web" ;;
    esac
  fi
  exit 0
fi

if [ -n "$tool" ] && [ -f "$allowlist" ]; then
  while IFS= read -r pattern || [ -n "$pattern" ]; do
    case "$pattern" in
      '' | '#'*) continue ;;
    esac
    if printf '%s\n' "$tool" | grep -E -q -x -- "$pattern"; then
      exit 0
    fi
  done < "$allowlist"
fi

block "${tool:-<sem nome>}"
