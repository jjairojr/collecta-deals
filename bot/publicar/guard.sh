#!/bin/sh
here="$(cd "$(dirname "$0")" && pwd)"
allowlist="$here/ferramentas-permitidas.txt"
logdir="${PUBLICAR_DADOS:-/Users/Shared/collecta-bot/publicar}"
approved="$logdir/aprovados"
input="$(cat)"

field() {
  printf '%s' "$input" | /usr/bin/plutil -extract "$1" raw -o - - 2>/dev/null
}

block() {
  mkdir -p "$logdir"
  printf '%s bloqueado: %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$1" >> "$logdir/bloqueios.log"
  echo "Bloqueado no Publicador: $1. Use só o Instagram, o TikTok Studio, os arquivos de aprovados e as ferramentas da collecta-publicar." >&2
  exit 2
}

allowed_url() {
  case "$1" in
    https://*) ;;
    *) return 1 ;;
  esac
  rest="${1#https://}"
  host="${rest%%[/?#]*}"
  path="/${rest#"$host"}"
  path="${path#/}"
  case "$host" in
    *@* | *:*) return 1 ;;
  esac
  host="$(printf '%s' "$host" | tr '[:upper:]' '[:lower:]')"
  case "$path" in
    /direct* | /accounts* | /ads* | /messages* | /settings* | /business* | /live*) return 1 ;;
  esac
  case "$host" in
    instagram.com | www.instagram.com) return 0 ;;
    tiktok.com | www.tiktok.com) return 0 ;;
  esac
  return 1
}

approved_paths() {
  i=0
  while :; do
    p="$(field "tool_input.arguments.paths.$i")"
    [ -z "$p" ] && break
    case "$p" in
      *..*) return 1 ;;
      "$approved"/*/*) ;;
      *) return 1 ;;
    esac
    i=$((i + 1))
  done
  [ "$i" -gt 0 ]
}

tool="$(field tool_name)"

if [ "$tool" = "mcp__openbot__tool_call" ]; then
  target="$(field tool_input.name)"
  case "$target" in
    openbot_browser.upload_files) approved_paths || block "upload_files fora da pasta de aprovados" ;;
    openbot_browser.run_script | openbot_browser.evaluate | openbot_browser.fill_fields) block "$target" ;;
    openbot_browser.*login* | openbot_browser.*secret* | openbot_browser.*download* | openbot_browser.*vault* | openbot_browser.*password*) block "$target" ;;
    openbot_browser.*) ;;
    *) block "tool_call ${target:-sem nome}" ;;
  esac
  url="$(field tool_input.arguments.url)"
  if [ -n "$url" ] && ! allowed_url "$url"; then
    block "$target fora do Instagram e do TikTok Studio ($url)"
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
