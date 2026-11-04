package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	errOpenBotOff  = errors.New("OpenBot fechado ou Local scripts desligado (sem a pasta automation)")
	errRateLimited = errors.New("OpenBot recusou: limite de 30 rodadas por hora")
	safeID         = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
)

type openBot struct {
	dir     string
	agent   string
	routine string
	client  *http.Client
}

func readTrimmed(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func (o openBot) fire(ctx context.Context, payload string) (string, error) {
	base, err := readTrimmed(filepath.Join(o.dir, "automation", "url"))
	if err != nil || base == "" {
		return "", errOpenBotOff
	}
	token, err := readTrimmed(filepath.Join(o.dir, "automation", "token"))
	if err != nil || token == "" {
		return "", errOpenBotOff
	}
	body, err := json.Marshal(map[string]string{"payload": payload})
	if err != nil {
		return "", err
	}
	url := strings.TrimRight(base, "/") + "/v1/agents/" + o.agent + "/routines/" + o.routine + "/run"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("OpenBot não respondeu: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	switch resp.StatusCode {
	case http.StatusAccepted, http.StatusOK:
	case http.StatusTooManyRequests:
		return "", errRateLimited
	default:
		return "", fmt.Errorf("OpenBot respondeu %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out struct {
		RunID string `json:"runId"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("resposta do OpenBot ilegível: %w", err)
	}
	return out.RunID, nil
}

func (o openBot) recentRuns() ([]run, error) {
	if !safeID.MatchString(o.agent) {
		return nil, fmt.Errorf("id de agente inválido: %q", o.agent)
	}
	db := filepath.Join(o.dir, "openbot.db")
	query := "select status, created_at, coalesce(run_source, '') from projection_routine_runs where agent_id = '" + o.agent + "' order by created_at desc limit 10"
	out, err := exec.Command("sqlite3", "-readonly", "-separator", "|", db, query).Output()
	if err != nil {
		return nil, fmt.Errorf("ler rodadas do OpenBot: %w", err)
	}
	return parseRuns(string(out))
}

func parseRuns(out string) ([]run, error) {
	var runs []run
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "|")
		if len(fields) != 3 {
			return nil, fmt.Errorf("linha inesperada do OpenBot: %q", line)
		}
		t, err := time.Parse(time.RFC3339Nano, fields[1])
		if err != nil {
			return nil, fmt.Errorf("data inesperada do OpenBot: %q", fields[1])
		}
		runs = append(runs, run{Status: fields[0], CreatedAt: t, Source: fields[2]})
	}
	return runs, nil
}
