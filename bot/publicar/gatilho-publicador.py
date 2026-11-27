#!/usr/bin/env python3
"""Acorda o Publicador Collecta só quando há post aprovado no horário.

Economia de limite da IA (curso "Exército de Bots", aulas 16 e 17): em vez de o
Publicador rodar de hora em hora mesmo sem nada, este script, sem IA, olha a fila
da loja a cada 10 minutos (launchd com.collecta.publicador-gatilho) e só dispara a
rotina do Publicador quando existe item aprovado cujo horário já chegou.
A rotina do Publicador continua com poucas rodadas fixas, como rede de segurança.

uso: gatilho-publicador.py            (roda de verdade)
     gatilho-publicador.py --teste    (só mostra o que faria)
"""
import json
import os
import sys
import time
import urllib.error
import urllib.request
from datetime import datetime, timezone

DADOS = "/Users/Shared/collecta-bot/publicar"
ENV = os.path.join(DADOS, "anuncios.env")
ESTADO = os.path.join(DADOS, "gatilho-publicador.json")
LOG = os.path.join(DADOS, "gatilho-publicador.log")
PAINEL = "https://collecta-deals-production.up.railway.app"
AUTOMACAO = os.path.expanduser("~/Library/Application Support/OpenBot/automation")
PUBLICADOR = "agent-863e931f-35bc-45c7-9e59-6329655cebaa"
ROTINA = "a83daf1a-f061-4eb1-b5cd-7cab13c9b08a"
INTERVALO_MIN = 25 * 60  # não acorda de novo antes disso, salvo item novo


def log(msg):
    with open(LOG, "a") as f:
        f.write(f"{datetime.now().strftime('%Y-%m-%d %H:%M')} {msg}\n")


def env_token():
    for line in open(ENV):
        if line.startswith("OPDEALS_ADMIN_TOKEN="):
            return line.split("=", 1)[1].strip().strip('"')
    raise SystemExit("OPDEALS_ADMIN_TOKEN não está em anuncios.env")


def aprovados_no_horario():
    req = urllib.request.Request(PAINEL + "/api/publish?status=approved",
                                 headers={"X-Admin-Token": env_token(), "User-Agent": "collecta-gatilho"})
    itens = json.load(urllib.request.urlopen(req, timeout=30)).get("items") or []
    agora = datetime.now(timezone.utc)
    devidos = []
    for it in itens:
        quando = datetime.fromisoformat(it["postAt"].replace("Z", "+00:00"))
        if quando <= agora:
            devidos.append(it["id"])
    return sorted(devidos)


def acordar(devidos):
    url = open(os.path.join(AUTOMACAO, "url")).read().strip()
    headers = {"Content-Type": "application/json"}
    for line in open(os.path.join(AUTOMACAO, "headers")):
        if ":" in line:
            k, v = line.split(":", 1)
            headers[k.strip()] = v.strip()
    corpo = json.dumps({"payload": "Gatilho sem IA: posts aprovados no horário: " + ", ".join(devidos)}).encode()
    req = urllib.request.Request(f"{url}/v1/agents/{PUBLICADOR}/routines/{ROTINA}/run", data=corpo, headers=headers, method="POST")
    with urllib.request.urlopen(req, timeout=30) as r:
        return r.status


def main():
    teste = "--teste" in sys.argv
    hora = datetime.now().hour
    if not 9 <= hora <= 22:
        return
    devidos = aprovados_no_horario()
    estado = {}
    if os.path.exists(ESTADO):
        estado = json.load(open(ESTADO))
    if not devidos:
        if teste:
            print("nada aprovado no horário")
        return
    novos = [d for d in devidos if d not in estado.get("ids", [])]
    passou = time.time() - estado.get("quando", 0) >= INTERVALO_MIN
    if not novos and not passou:
        if teste:
            print("já acordado há pouco para", devidos)
        return
    if teste:
        print("acordaria o Publicador para", devidos)
        return
    status = acordar(devidos)
    json.dump({"ids": devidos, "quando": time.time()}, open(ESTADO, "w"))
    log(f"acordou o Publicador ({status}) para {', '.join(devidos)}")


if __name__ == "__main__":
    try:
        main()
    except (urllib.error.URLError, TimeoutError, ConnectionError) as e:
        # Rede ou DNS fora por um instante: uma linha com horário e sai sem erro.
        # A próxima rodada (10 min) tenta de novo sozinha. Sem traceback, para o
        # log não parecer uma sequência de falhas sem horário.
        motivo = getattr(e, "reason", e)
        log(f"falha de rede, tenta de novo na próxima rodada: {motivo}")
    except Exception as e:  # noqa: BLE001
        log(f"erro: {e}")
        raise
