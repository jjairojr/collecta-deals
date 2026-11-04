package main

import (
	"context"
	"fmt"
	"time"
)

const alertEvery = time.Hour

type vigia struct {
	cfg     config
	now     func() time.Time
	pending func(context.Context) (pendingWork, error)
	runs    func() ([]run, error)
	fire    func(context.Context, string) (string, error)
	notify  func(string, string)
	logf    func(string, ...any)

	lastFire     time.Time
	lastFull     time.Time
	failingSince time.Time
	alertedAt    time.Time
	problems     map[string]string
}

func (v *vigia) loop(ctx context.Context) {
	v.step(ctx)
	ticker := time.NewTicker(v.cfg.Tick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			v.step(ctx)
		}
	}
}

func (v *vigia) report(source string, err error) {
	if v.problems == nil {
		v.problems = map[string]string{}
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	if v.problems[source] == msg {
		return
	}
	v.problems[source] = msg
	if msg == "" {
		v.logf("%s: voltou ao normal", source)
		return
	}
	v.logf("%s: %s", source, msg)
}

func (v *vigia) step(ctx context.Context) {
	now := v.now()
	if !v.cfg.inHours(now) {
		v.failingSince = time.Time{}
		return
	}
	obs := observation{Now: now, LastRun: v.lastFire, LastFull: v.lastFull}
	runs, err := v.runs()
	v.report("rodadas do OpenBot", err)
	if err == nil {
		last, full, busy := summarizeRuns(runs, now)
		if last.After(obs.LastRun) {
			obs.LastRun = last
		}
		if full.After(obs.LastFull) {
			obs.LastFull = full
		}
		obs.Busy = busy
	}
	pctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	obs.Pending, err = v.pending(pctx)
	cancel()
	obs.PendingOK = err == nil
	v.report("pós-venda", err)
	d := decide(v.cfg, obs)
	if !d.Fire {
		return
	}
	fctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	runID, err := v.fire(fctx, d.Payload)
	cancel()
	if err != nil {
		v.fireFailed(now, d.Reason, err)
		return
	}
	v.lastFire = now
	if d.Full {
		v.lastFull = now
	}
	v.failingSince = time.Time{}
	v.report("disparo", nil)
	v.logf("acordou o atendente (%s) rodada %s", d.Reason, runID)
}

func (v *vigia) fireFailed(now time.Time, reason string, err error) {
	if v.failingSince.IsZero() {
		v.failingSince = now
	}
	v.report("disparo", fmt.Errorf("não acordou o atendente (%s): %w", reason, err))
	if now.Sub(v.failingSince) < v.cfg.AlertAfter || (!v.alertedAt.IsZero() && now.Sub(v.alertedAt) < alertEvery) {
		return
	}
	v.alertedAt = now
	v.notify("Vigia do atendente parado", fmt.Sprintf("Há %d min sem conseguir acordar o atendente: %v", int(now.Sub(v.failingSince).Minutes()), err))
}
