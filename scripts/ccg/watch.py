import gzip
import json
import re
import subprocess
import sys
import time
import urllib.request

LIST_URL = "https://www.ccgeventos.com.br/ingressos"
EVENT_URL = "https://www.ccgeventos.com.br/informa-es-do-evento-e-registro/"
UA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0 Safari/537.36"
STATUS = {0: "SCHEDULED", 1: "CLOSED", 2: "CLOSED_MANUALLY", 3: "OPEN_RSVP",
          4: "OPEN_RSVP_WAITLIST", 5: "OPEN_TICKETS", 6: "OPEN_EXTERNAL"}
OPEN = {3, 4, 5, 6}


def fetch(url):
    req = urllib.request.Request(url, headers={
        "User-Agent": UA,
        "Cache-Control": "no-cache",
        "Accept-Encoding": "gzip",
    })
    with urllib.request.urlopen(req, timeout=30) as r:
        raw = r.read()
        if r.headers.get("Content-Encoding") == "gzip":
            raw = gzip.decompress(raw)
        return raw.decode("utf-8", "replace")


def scan(html):
    slugs = [(m.start(), m.end(), m.group(1)) for m in re.finditer(r'"slug":"([^"]+)"', html)]
    out = {}
    for i, (pos, end, slug) in enumerate(slugs):
        if slug in out:
            continue
        stop = slugs[i + 1][0] if i + 1 < len(slugs) else len(html)
        fwd = html[end:stop]
        reg = re.search(r'"registration":\{"type":(\d+),"status":(\d+)', fwd)
        if not reg:
            continue
        sold = re.search(r'"soldOut":(true|false)', fwd)
        back_from = slugs[i - 1][1] if i else max(0, pos - 6000)
        titles = re.findall(r'"title":"((?:[^"\\]|\\.)*)"', html[back_from:pos])
        out[slug] = {
            "title": json.loads('"%s"' % titles[-1]) if titles else slug,
            "status": int(reg.group(2)),
            "soldOut": sold.group(1) == "true" if sold else None,
        }
    return out


def notify(title, msg):
    subprocess.run(["osascript", "-e",
                    'display notification %s with title %s sound name "Glass"'
                    % (json.dumps(msg), json.dumps(title))], check=False)


def alarm(title, msg, slug=None):
    notify(title, msg)
    if slug:
        subprocess.run(["open", EVENT_URL + slug], check=False)
    for _ in range(6):
        subprocess.run(["afplay", "-v", "3", "/System/Library/Sounds/Sosumi.aiff"], check=False)
    subprocess.run(["say", "-v", "Luciana", "Ingresso aberto. " + msg], check=False)
    subprocess.run(["osascript", "-e",
                    'display alert %s message %s as critical'
                    % (json.dumps(title), json.dumps(msg))], check=False)


def main():
    argv = sys.argv[1:]
    interval = 10
    if "-i" in argv:
        k = argv.index("-i")
        interval = int(argv[k + 1])
        del argv[k:k + 2]
    targets = argv
    print("%s  poll a cada %ds | alvos: %s" % (time.strftime("%H:%M:%S"), interval, targets or "todos"), flush=True)
    prev = {}
    while True:
        try:
            events = scan(fetch(LIST_URL))
        except Exception as e:
            print("%s  erro: %s" % (time.strftime("%H:%M:%S"), e), flush=True)
            time.sleep(interval)
            continue
        for slug, ev in events.items():
            if targets and not any(t in slug for t in targets):
                continue
            cur = (ev["status"], ev["soldOut"])
            old = prev.get(slug)
            label = "%s status=%s soldOut=%s" % (slug, STATUS.get(ev["status"], ev["status"]), ev["soldOut"])
            if old is None:
                print("%s  base: %s" % (time.strftime("%H:%M:%S"), label), flush=True)
            elif old != cur:
                print("%s  MUDOU: %s (antes status=%s soldOut=%s)"
                      % (time.strftime("%H:%M:%S"), label, STATUS.get(old[0], old[0]), old[1]), flush=True)
                if ev["status"] in OPEN and not ev["soldOut"]:
                    alarm("INGRESSO ABERTO", ev["title"], slug)
                else:
                    notify("Evento mudou", "%s: %s" % (ev["title"], STATUS.get(ev["status"], ev["status"])))
            prev[slug] = cur
        time.sleep(interval)


main()
