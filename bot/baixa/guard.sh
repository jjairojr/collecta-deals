#!/bin/sh
here="$(cd "$(dirname "$0")" && pwd)"
allowlist="$here/ferramentas-permitidas.txt"
logdir="${BAIXA_DADOS:-$HOME/.collecta-baixa}"
input="$(cat)"

field() {
  printf '%s' "$input" | /usr/bin/plutil -extract "$1" raw -o - - 2>/dev/null
}

block() {
  mkdir -p "$logdir"
  printf '%s bloqueado: %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$1" >> "$logdir/bloqueios.log"
  echo "Bloqueado na baixa: $1. Use só o navegador na Liga e as ferramentas da collecta-baixa." >&2
  exit 2
}

allowed_url() {
  case "$1" in
    https://*) ;;
    *) return 1 ;;
  esac
  rest="${1#https://}"
  host="${rest%%[/?#]*}"
  case "$host" in
    *@* | *:*) return 1 ;;
  esac
  host="$(printf '%s' "$host" | tr '[:upper:]' '[:lower:]')"
  case "$host" in
    ligamagic.com.br | *.ligamagic.com.br) return 0 ;;
    ligacollectatcg.com.br | *.ligacollectatcg.com.br) return 0 ;;
    ligaonepiece.com.br | *.ligaonepiece.com.br) return 0 ;;
    ligapokemon.com.br | *.ligapokemon.com.br) return 0 ;;
    ligariftbound.com.br | *.ligariftbound.com.br) return 0 ;;
  esac
  return 1
}

tool="$(field tool_name)"
args="tool_input.arguments"
target=""
case "$tool" in
  mcp__openbot__tool_call) target="$(field tool_input.name)" ;;
  mcp__openbot_browser__*) target="openbot_browser.${tool#mcp__openbot_browser__}"; args="tool_input" ;;
esac

if [ "$tool" = "mcp__openbot__tool_call" ] || [ -n "$target" ]; then
  case "$target" in
    openbot_browser.upload_files | openbot_browser.fill_fields) block "$target" ;;
    openbot_browser.*login* | openbot_browser.*secret* | openbot_browser.*download* | openbot_browser.*vault* | openbot_browser.*password*) block "$target" ;;
    openbot_browser.*) ;;
    *) block "tool_call ${target:-sem nome}" ;;
  esac
  url="$(field "$args.url")"
  if [ -n "$url" ] && ! allowed_url "$url"; then
    block "$target fora da Liga"
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
