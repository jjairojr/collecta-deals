package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testConfig() config {
	return config{
		OpdealsURL: "http://opdeals.test",
		AgentToken: "agent",
		AgentID:    "agent-1",
		RoutineID:  "routine-1",
		Heartbeat:  20 * time.Minute,
		OpenEvery:  5 * time.Minute,
		MinGap:     3 * time.Minute,
		AlertAfter: 15 * time.Minute,
		Tick:       time.Minute,
		StartHour:  8,
		EndHour:    22,
	}
}

func at(hour, minute int) time.Time {
	return time.Date(2026, 11, 4, hour, minute, 0, 0, saoPaulo)
}

func TestDecide(t *testing.T) {
	cfg := testConfig()
	noon := at(12, 0)
	recentFull := noon.Add(-10 * time.Minute)
	cases := []struct {
		name   string
		obs    observation
		fire   bool
		full   bool
		reason string
	}{
		{"before hours", observation{Now: at(7, 59), PendingOK: true, Pending: pendingWork{Compose: 1}}, false, false, "fora do horário"},
		{"after hours", observation{Now: at(22, 0)}, false, false, "fora do horário"},
		{"first run of the day", observation{Now: at(8, 0), PendingOK: true}, true, true, "rodada completa, batimento"},
		{"busy", observation{Now: noon, Busy: true, PendingOK: true, Pending: pendingWork{Outbox: 2}}, false, false, "rodada em andamento"},
		{"recent run blocks pending", observation{Now: noon, LastRun: noon.Add(-2 * time.Minute), LastFull: recentFull, PendingOK: true, Pending: pendingWork{Compose: 1}}, false, false, "rodada recente"},
		{"pending only", observation{Now: noon, LastRun: noon.Add(-3 * time.Minute), LastFull: recentFull, PendingOK: true, Pending: pendingWork{Compose: 1, Outbox: 2, Tracking: 1}}, true, false, "só pós-venda, pós-venda: 1 primeira mensagem, 2 mensagens na fila, 1 rastreio"},
		{"pending when a full round is due", observation{Now: noon, LastRun: noon.Add(-3 * time.Minute), LastFull: noon.Add(-20 * time.Minute), PendingOK: true, Pending: pendingWork{Outbox: 1}}, true, true, "rodada completa, pós-venda: 1 mensagem na fila"},
		{"open waits for its interval", observation{Now: noon, LastRun: noon.Add(-4 * time.Minute), LastFull: recentFull, PendingOK: true, Pending: pendingWork{Open: 1}}, false, false, "nada a fazer"},
		{"open only", observation{Now: noon, LastRun: noon.Add(-5 * time.Minute), LastFull: recentFull, PendingOK: true, Pending: pendingWork{Open: 2}}, true, false, "só pós-venda, pós-venda: 2 conversas abertas"},
		{"open rounds cannot starve the full round", observation{Now: noon, LastRun: noon.Add(-5 * time.Minute), LastFull: noon.Add(-21 * time.Minute), PendingOK: true, Pending: pendingWork{Open: 2}}, true, true, "rodada completa, pós-venda: 2 conversas abertas"},
		{"idle before heartbeat", observation{Now: noon, LastRun: noon.Add(-19 * time.Minute), LastFull: noon.Add(-19 * time.Minute), PendingOK: true}, false, false, "nada a fazer"},
		{"heartbeat counts from the last full round", observation{Now: noon, LastRun: noon.Add(-4 * time.Minute), LastFull: noon.Add(-20 * time.Minute), PendingOK: true}, true, true, "rodada completa, batimento"},
		{"server down still heartbeats", observation{Now: noon, LastRun: noon.Add(-25 * time.Minute), LastFull: noon.Add(-25 * time.Minute)}, true, true, "rodada completa, batimento"},
		{"server down ignores stale numbers", observation{Now: noon, LastRun: noon.Add(-10 * time.Minute), LastFull: recentFull, Pending: pendingWork{Compose: 3}}, false, false, "nada a fazer"},
	}
	for _, c := range cases {
		d := decide(cfg, c.obs)
		if d.Fire != c.fire || d.Full != c.full || d.Reason != c.reason {
			t.Errorf("%s: got %+v", c.name, d)
		}
		switch {
		case d.Fire && d.Full && !strings.HasPrefix(d.Payload, "Vigia: rodada completa ("):
			t.Errorf("%s: payload %q", c.name, d.Payload)
		case d.Fire && !d.Full && !strings.HasPrefix(d.Payload, "Vigia: só pós-venda ("):
			t.Errorf("%s: payload %q", c.name, d.Payload)
		}
	}
}

func TestSummarizeRuns(t *testing.T) {
	now := at(12, 0)
	last, full, busy := summarizeRuns([]run{
		{Status: "succeeded", CreatedAt: now.Add(-10 * time.Minute), Source: ""},
		{Status: "succeeded", CreatedAt: now.Add(-6 * time.Minute), Source: "script"},
		{Status: "running", CreatedAt: now.Add(-2 * time.Minute), Source: "script"},
	}, now)
	if !busy || !last.Equal(now.Add(-2*time.Minute)) || !full.Equal(now.Add(-10*time.Minute)) {
		t.Fatalf("running: %v %v %v", last, full, busy)
	}
	if _, _, busy := summarizeRuns([]run{{Status: "running", CreatedAt: now.Add(-31 * time.Minute)}}, now); busy {
		t.Fatal("a run stuck for 31 min still blocks the watcher")
	}
	if last, full, busy := summarizeRuns(nil, now); busy || !last.IsZero() || !full.IsZero() {
		t.Fatalf("empty: %v %v %v", last, full, busy)
	}
}

func TestParseRuns(t *testing.T) {
	runs, err := parseRuns("running|2026-11-04T14:15:54.066Z|script\nsucceeded|2026-11-04T14:15:00.045Z|\n")
	if err != nil || len(runs) != 2 || runs[0].Status != "running" || runs[0].Source != "script" || runs[1].Source != "" || runs[0].CreatedAt.Minute() != 15 {
		t.Fatalf("%+v %v", runs, err)
	}
	if runs, err := parseRuns(""); err != nil || len(runs) != 0 {
		t.Fatalf("empty %+v %v", runs, err)
	}
	if _, err := parseRuns("garbage"); err == nil {
		t.Fatal("garbage accepted")
	}
}

func writeAutomation(t *testing.T, url string) string {
	t.Helper()
	dir := t.TempDir()
	auto := filepath.Join(dir, "automation")
	if err := os.MkdirAll(auto, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"url": url + "\n", "token": "local-token\n"} {
		if err := os.WriteFile(filepath.Join(auto, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestOpenBotFire(t *testing.T) {
	var gotPath, gotAuth, gotPayload string
	status := http.StatusAccepted
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		var in map[string]string
		_ = json.NewDecoder(r.Body).Decode(&in)
		gotPayload = in["payload"]
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"runId":"run-9","deliveryId":"d"}`))
	}))
	defer srv.Close()
	bot := openBot{dir: writeAutomation(t, srv.URL), agent: "agent-1", routine: "routine-1", client: srv.Client()}
	id, err := bot.fire(context.Background(), "Vigia: batimento.")
	if err != nil || id != "run-9" {
		t.Fatalf("fire %q %v", id, err)
	}
	if gotPath != "/v1/agents/agent-1/routines/routine-1/run" || gotAuth != "Bearer local-token" || gotPayload != "Vigia: batimento." {
		t.Fatalf("request %s %s %s", gotPath, gotAuth, gotPayload)
	}
	status = http.StatusTooManyRequests
	if _, err := bot.fire(context.Background(), "x"); !errors.Is(err, errRateLimited) {
		t.Fatalf("429: %v", err)
	}
	status = http.StatusForbidden
	if _, err := bot.fire(context.Background(), "x"); err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("403: %v", err)
	}
	closed := openBot{dir: t.TempDir(), agent: "agent-1", routine: "routine-1", client: srv.Client()}
	if _, err := closed.fire(context.Background(), "x"); !errors.Is(err, errOpenBotOff) {
		t.Fatalf("closed: %v", err)
	}
}

func TestFetchPending(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/postsale/agent/pending" || r.Header.Get("Authorization") != "Bearer agent" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"quiet":false,"compose":1,"outbox":0,"tracking":2,"open":1,"oldestAt":"2026-11-04T14:00:00Z"}`))
	}))
	defer srv.Close()
	cfg := testConfig()
	cfg.OpdealsURL = srv.URL
	p, err := fetchPending(context.Background(), srv.Client(), cfg)
	if err != nil || p.Total() != 3 || p.Open != 1 {
		t.Fatalf("%+v %v", p, err)
	}
	cfg.AgentToken = "wrong"
	if _, err := fetchPending(context.Background(), srv.Client(), cfg); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("wrong token: %v", err)
	}
}

type fakeWorld struct {
	now      time.Time
	runs     []run
	runsErr  error
	pending  pendingWork
	pendErr  error
	fireErr  error
	fired    []string
	notified []string
	logs     []string
}

func (f *fakeWorld) vigia() *vigia {
	return &vigia{
		cfg:     testConfig(),
		now:     func() time.Time { return f.now },
		pending: func(context.Context) (pendingWork, error) { return f.pending, f.pendErr },
		runs:    func() ([]run, error) { return f.runs, f.runsErr },
		fire: func(_ context.Context, payload string) (string, error) {
			if f.fireErr != nil {
				return "", f.fireErr
			}
			f.fired = append(f.fired, payload)
			f.runs = append(f.runs, run{Status: "succeeded", CreatedAt: f.now, Source: "script"})
			return "run", nil
		},
		notify: func(title, _ string) { f.notified = append(f.notified, title) },
		logf:   func(format string, _ ...any) { f.logs = append(f.logs, format) },
	}
}

func TestVigiaStepsThroughTheDay(t *testing.T) {
	f := &fakeWorld{now: at(7, 58)}
	v := f.vigia()
	v.step(context.Background())
	if len(f.fired) != 0 {
		t.Fatal("fired before 8h")
	}
	f.now = at(8, 0)
	v.step(context.Background())
	f.now = at(8, 1)
	f.pending = pendingWork{Compose: 1}
	v.step(context.Background())
	if len(f.fired) != 1 || !strings.Contains(f.fired[0], "rodada completa") {
		t.Fatalf("first round of the day: %v", f.fired)
	}
	f.now = at(8, 3)
	v.step(context.Background())
	if len(f.fired) != 2 || !strings.Contains(f.fired[1], "só pós-venda (pós-venda: 1 primeira mensagem)") {
		t.Fatalf("pending not fired as post-sale only: %v", f.fired)
	}
	f.pending = pendingWork{Open: 1}
	for m := 4; m <= 19; m++ {
		f.now = at(8, m)
		v.step(context.Background())
	}
	for _, p := range f.fired[2:] {
		if !strings.Contains(p, "só pós-venda") {
			t.Fatalf("open conversation rounds should be post-sale only: %v", f.fired)
		}
	}
	before := len(f.fired)
	f.now = at(8, 20)
	v.step(context.Background())
	if len(f.fired) != before {
		t.Fatalf("full round ignored the minimum gap after the 8:18 round: %v", f.fired)
	}
	f.now = at(8, 21)
	v.step(context.Background())
	if len(f.fired) != before+1 || !strings.Contains(f.fired[before], "rodada completa (batimento)") {
		t.Fatalf("no full round once the gap allowed it: %v", f.fired)
	}
}

func TestVigiaCountsCronRuns(t *testing.T) {
	f := &fakeWorld{now: at(10, 0), runs: []run{{Status: "succeeded", CreatedAt: at(9, 55)}}}
	f.pending = pendingWork{}
	v := f.vigia()
	v.step(context.Background())
	if len(f.fired) != 0 {
		t.Fatalf("ignored a cron run 5 min ago: %v", f.fired)
	}
	f.runs = []run{{Status: "running", CreatedAt: at(9, 45)}}
	f.pending = pendingWork{Outbox: 1}
	v.step(context.Background())
	if len(f.fired) != 0 {
		t.Fatalf("stacked a run on top of a running one: %v", f.fired)
	}
}

func TestVigiaAlertsAfterFifteenMinutesFailing(t *testing.T) {
	f := &fakeWorld{now: at(9, 0), fireErr: errOpenBotOff}
	v := f.vigia()
	for m := 0; m <= 14; m++ {
		f.now = at(9, m)
		v.step(context.Background())
	}
	if len(f.notified) != 0 {
		t.Fatalf("alerted before 15 min: %v", f.notified)
	}
	if len(f.logs) != 1 {
		t.Fatalf("repeated the same failure in the log: %v", f.logs)
	}
	f.now = at(9, 15)
	v.step(context.Background())
	f.now = at(9, 40)
	v.step(context.Background())
	if len(f.notified) != 1 {
		t.Fatalf("alerts %v", f.notified)
	}
	f.fireErr = nil
	f.now = at(9, 41)
	v.step(context.Background())
	if len(f.fired) != 1 || !v.failingSince.IsZero() {
		t.Fatalf("did not recover: %v", f.fired)
	}
}

func TestLoadConfig(t *testing.T) {
	env := map[string]string{"POSTSALE_AGENT_TOKEN": "tok", "NTFY_TOPIC": "topico", "VIGIA_BATIMENTO": "15m", "VIGIA_HORARIO": "9-21"}
	cfg, err := loadConfig(func(k string) string { return env[k] }, "/home/x")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Heartbeat != 15*time.Minute || cfg.OpenEvery != 5*time.Minute || cfg.StartHour != 9 || cfg.EndHour != 21 || cfg.NtfyURL != "https://ntfy.sh/topico" || cfg.OpenBotDir != "/home/x/Library/Application Support/OpenBot" {
		t.Fatalf("%+v", cfg)
	}
	for name, bad := range map[string]map[string]string{
		"no token":  {},
		"bad hours": {"POSTSALE_AGENT_TOKEN": "tok", "VIGIA_HORARIO": "22-8"},
		"bad beat":  {"POSTSALE_AGENT_TOKEN": "tok", "VIGIA_BATIMENTO": "logo"},
		"bad agent": {"POSTSALE_AGENT_TOKEN": "tok", "OPENBOT_AGENTE": "a'; drop"},
	} {
		if _, err := loadConfig(func(k string) string { return bad[k] }, "/home/x"); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
}

func TestLoadEnvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vigia.env")
	if err := os.WriteFile(path, []byte("# comentário\nPOSTSALE_AGENT_TOKEN=\"abc\"\n\nNTFY_TOPIC = topico\nlixo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env, err := loadEnvFile(path)
	if err != nil || env["POSTSALE_AGENT_TOKEN"] != "abc" || env["NTFY_TOPIC"] != "topico" || len(env) != 2 {
		t.Fatalf("%v %v", env, err)
	}
	if env, err := loadEnvFile(filepath.Join(t.TempDir(), "nada.env")); err != nil || len(env) != 0 {
		t.Fatalf("missing file %v %v", env, err)
	}
}
