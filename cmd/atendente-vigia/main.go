package main

import (
	"bufio"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata"
)

var saoPaulo = mustLoadLocation("America/Sao_Paulo")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

type config struct {
	OpdealsURL string
	AgentToken string
	NtfyURL    string
	OpenBotDir string
	AgentID    string
	RoutineID  string
	Heartbeat  time.Duration
	OpenEvery  time.Duration
	MinGap     time.Duration
	AlertAfter time.Duration
	Tick       time.Duration
	StartHour  int
	EndHour    int
}

func loadEnvFile(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	return out, sc.Err()
}

func durationEnv(getenv func(string) string, key string, def time.Duration) (time.Duration, error) {
	raw := getenv(key)
	if raw == "" {
		return def, nil
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s inválido: %q (use por exemplo 20m)", key, raw)
	}
	return d, nil
}

func parseHours(raw string) (int, int, error) {
	from, to, ok := strings.Cut(raw, "-")
	start, err1 := strconv.Atoi(strings.TrimSpace(from))
	end, err2 := strconv.Atoi(strings.TrimSpace(to))
	if !ok || err1 != nil || err2 != nil || start < 0 || end > 24 || start >= end {
		return 0, 0, fmt.Errorf("VIGIA_HORARIO inválido: %q (use por exemplo 8-22)", raw)
	}
	return start, end, nil
}

func loadConfig(getenv func(string) string, home string) (config, error) {
	cfg := config{
		OpdealsURL: strings.TrimRight(cmp.Or(getenv("OPDEALS_URL"), "https://collecta-deals-production.up.railway.app"), "/"),
		AgentToken: getenv("POSTSALE_AGENT_TOKEN"),
		OpenBotDir: cmp.Or(getenv("OPENBOT_DIR"), filepath.Join(home, "Library", "Application Support", "OpenBot")),
		AgentID:    cmp.Or(getenv("OPENBOT_AGENTE"), "agent-d9a81719-259c-4cb0-b60b-c1bdec9ef8df"),
		RoutineID:  cmp.Or(getenv("OPENBOT_ROTINA"), "76cadb3e-90c6-4557-a014-d683a0f5d7ac"),
		AlertAfter: 15 * time.Minute,
		Tick:       time.Minute,
	}
	switch {
	case getenv("NTFY_URL") != "":
		cfg.NtfyURL = getenv("NTFY_URL")
	case getenv("NTFY_TOPIC") != "":
		cfg.NtfyURL = "https://ntfy.sh/" + getenv("NTFY_TOPIC")
	}
	if cfg.AgentToken == "" {
		return cfg, errors.New("POSTSALE_AGENT_TOKEN não definido: sem ele o vigia não enxerga o pós-venda")
	}
	if !safeID.MatchString(cfg.AgentID) || !safeID.MatchString(cfg.RoutineID) {
		return cfg, errors.New("OPENBOT_AGENTE ou OPENBOT_ROTINA com caracteres inválidos")
	}
	var err error
	if cfg.Heartbeat, err = durationEnv(getenv, "VIGIA_BATIMENTO", 20*time.Minute); err != nil {
		return cfg, err
	}
	if cfg.OpenEvery, err = durationEnv(getenv, "VIGIA_ABERTAS", 5*time.Minute); err != nil {
		return cfg, err
	}
	if cfg.MinGap, err = durationEnv(getenv, "VIGIA_INTERVALO_MINIMO", 3*time.Minute); err != nil {
		return cfg, err
	}
	if cfg.StartHour, cfg.EndHour, err = parseHours(cmp.Or(getenv("VIGIA_HORARIO"), "8-22")); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func fetchPending(ctx context.Context, client *http.Client, cfg config) (pendingWork, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.OpdealsURL+"/api/postsale/agent/pending", nil)
	if err != nil {
		return pendingWork{}, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.AgentToken)
	resp, err := client.Do(req)
	if err != nil {
		return pendingWork{}, fmt.Errorf("servidor do pós-venda não respondeu: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if resp.StatusCode != http.StatusOK {
		return pendingWork{}, fmt.Errorf("servidor do pós-venda respondeu %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var p pendingWork
	if err := json.Unmarshal(raw, &p); err != nil {
		return pendingWork{}, fmt.Errorf("resposta do pós-venda ilegível: %w", err)
	}
	return p, nil
}

func ntfy(client *http.Client, url string) func(string, string) {
	return func(title, msg string) {
		if url == "" {
			return
		}
		req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(msg))
		if err != nil {
			return
		}
		req.Header.Set("Title", title)
		req.Header.Set("Tags", "warning")
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("ntfy: %v", err)
			return
		}
		resp.Body.Close()
	}
}

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	envPath := cmp.Or(os.Getenv("VIGIA_ENV"), filepath.Join(home, ".collecta-atendente", "vigia.env"))
	file, err := loadEnvFile(envPath)
	if err != nil {
		log.Fatalf("ler %s: %v", envPath, err)
	}
	getenv := func(k string) string { return cmp.Or(os.Getenv(k), file[k]) }
	cfg, err := loadConfig(getenv, home)
	if err != nil {
		log.Fatal(err)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	bot := openBot{dir: cfg.OpenBotDir, agent: cfg.AgentID, routine: cfg.RoutineID, client: client}
	v := &vigia{
		cfg:     cfg,
		now:     time.Now,
		pending: func(ctx context.Context) (pendingWork, error) { return fetchPending(ctx, client, cfg) },
		runs:    bot.recentRuns,
		fire:    bot.fire,
		notify:  ntfy(client, cfg.NtfyURL),
		logf:    log.Printf,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Printf("vigia no ar: batimento %s, conversas abertas %s, intervalo mínimo %s, das %dh às %dh", cfg.Heartbeat, cfg.OpenEvery, cfg.MinGap, cfg.StartHour, cfg.EndHour)
	v.loop(ctx)
}
