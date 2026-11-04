package main

import (
	"fmt"
	"strings"
	"time"
)

type pendingWork struct {
	Quiet    bool      `json:"quiet"`
	Compose  int       `json:"compose"`
	Outbox   int       `json:"outbox"`
	Tracking int       `json:"tracking"`
	Open     int       `json:"open"`
	OldestAt time.Time `json:"oldestAt"`
}

func (p pendingWork) Total() int {
	return p.Compose + p.Outbox + p.Tracking
}

type run struct {
	Status    string
	CreatedAt time.Time
	Source    string
}

type observation struct {
	Now       time.Time
	LastRun   time.Time
	LastFull  time.Time
	Busy      bool
	Pending   pendingWork
	PendingOK bool
}

type decision struct {
	Fire    bool
	Full    bool
	Reason  string
	Payload string
}

const busyStaleAfter = 30 * time.Minute

func summarizeRuns(runs []run, now time.Time) (time.Time, time.Time, bool) {
	var last, full time.Time
	busy := false
	for _, r := range runs {
		if r.CreatedAt.After(last) {
			last = r.CreatedAt
		}
		if r.Source != "script" && r.CreatedAt.After(full) {
			full = r.CreatedAt
		}
		if (r.Status == "queued" || r.Status == "running") && now.Sub(r.CreatedAt) < busyStaleAfter {
			busy = true
		}
	}
	return last, full, busy
}

func (c config) inHours(t time.Time) bool {
	h := t.In(saoPaulo).Hour()
	return h >= c.StartHour && h < c.EndHour
}

func decide(c config, o observation) decision {
	if !c.inHours(o.Now) {
		return decision{Reason: "fora do horário"}
	}
	if o.Busy {
		return decision{Reason: "rodada em andamento"}
	}
	gap := o.Now.Sub(o.LastRun)
	if !o.LastRun.IsZero() && gap < c.MinGap {
		return decision{Reason: "rodada recente"}
	}
	needFull := o.LastFull.IsZero() || o.Now.Sub(o.LastFull) >= c.Heartbeat
	reason := ""
	switch {
	case o.PendingOK && o.Pending.Total() > 0:
		reason = "pós-venda: " + describePending(o.Pending)
	case o.PendingOK && o.Pending.Open > 0 && (o.LastRun.IsZero() || gap >= c.OpenEvery):
		reason = "pós-venda: " + plural(o.Pending.Open, "conversa aberta", "conversas abertas")
	case needFull:
		reason = "batimento"
	default:
		return decision{Reason: "nada a fazer"}
	}
	if needFull {
		return decision{Fire: true, Full: true, Reason: "rodada completa, " + reason, Payload: "Vigia: rodada completa (" + reason + ")."}
	}
	return decision{Fire: true, Reason: "só pós-venda, " + reason, Payload: "Vigia: só pós-venda (" + reason + "). Faça os passos 1 a 3 da rodada e encerre."}
}

func describePending(p pendingWork) string {
	var parts []string
	if p.Compose > 0 {
		parts = append(parts, plural(p.Compose, "primeira mensagem", "primeiras mensagens"))
	}
	if p.Outbox > 0 {
		parts = append(parts, plural(p.Outbox, "mensagem na fila", "mensagens na fila"))
	}
	if p.Tracking > 0 {
		parts = append(parts, plural(p.Tracking, "rastreio", "rastreios"))
	}
	return strings.Join(parts, ", ")
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}
