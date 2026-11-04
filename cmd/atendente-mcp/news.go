package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const dateLayout = "02/01/2006"

var dateLine = regexp.MustCompile(`(?i)^(desde|até|ate)\s*:\s*(\S+)\s*$`)

type newsItem struct {
	Titulo string `json:"titulo"`
	Texto  string `json:"texto"`
}

type newsOutput struct {
	Hoje      string     `json:"hoje"`
	Novidades []newsItem `json:"novidades"`
}

type newsEntry struct {
	item  newsItem
	from  time.Time
	until time.Time
	bad   bool
}

func parseNews(src string) []newsEntry {
	var entries []newsEntry
	var text []string
	flush := func() {
		if len(entries) == 0 {
			return
		}
		entries[len(entries)-1].item.Texto = strings.TrimSpace(strings.Join(text, "\n"))
		text = nil
	}
	scanner := bufio.NewScanner(strings.NewReader(src))
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " \t")
		if title, ok := strings.CutPrefix(line, "## "); ok {
			flush()
			entries = append(entries, newsEntry{item: newsItem{Titulo: strings.TrimSpace(title)}})
			continue
		}
		if len(entries) == 0 {
			continue
		}
		if m := dateLine.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			day, err := time.ParseInLocation(dateLayout, m[2], saoPaulo)
			current := &entries[len(entries)-1]
			if err != nil {
				current.bad = true
				continue
			}
			if strings.EqualFold(m[1], "desde") {
				current.from = day
			} else {
				current.until = day
			}
			continue
		}
		text = append(text, line)
	}
	flush()
	return entries
}

func activeNews(entries []newsEntry, now time.Time) []newsItem {
	local := now.In(saoPaulo)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, saoPaulo)
	out := []newsItem{}
	for _, e := range entries {
		if e.bad || e.item.Texto == "" {
			continue
		}
		if !e.from.IsZero() && today.Before(e.from) {
			continue
		}
		if !e.until.IsZero() && today.After(e.until) {
			continue
		}
		out = append(out, e.item)
	}
	return out
}

func (a *app) toolNews(_ context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, newsOutput, error) {
	out := newsOutput{Hoje: a.now().In(saoPaulo).Format(dateLayout), Novidades: []newsItem{}}
	data, err := os.ReadFile(a.cfg.NewsPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, out, nil
	}
	if err != nil {
		return nil, out, fmt.Errorf("não consegui ler as novidades: %v", err)
	}
	entries := parseNews(string(data))
	for _, e := range entries {
		if e.bad {
			log.Printf("novidades: data inválida em %q, item ignorado", e.item.Titulo)
		}
	}
	out.Novidades = activeNews(entries, a.now())
	return nil, out, nil
}

type originInput struct {
	Telefone string `json:"telefone" jsonschema:"telefone do cliente com DDD, ou ig:usuario no Instagram"`
	Origem   string `json:"origem" jsonschema:"como a pessoa achou a loja, nas palavras dela. Ex.: Instagram, Google, indicação de amigo"`
}

type originOutput struct {
	OK           bool `json:"ok"`
	JaRegistrado bool `json:"ja_registrado"`
}

func (a *app) toolOrigin(_ context.Context, _ *mcp.CallToolRequest, in originInput) (*mcp.CallToolResult, originOutput, error) {
	key, err := phoneKey(in.Telefone)
	if err != nil {
		return nil, originOutput{}, err
	}
	origin := strings.Join(strings.Fields(in.Origem), " ")
	if origin == "" {
		return nil, originOutput{}, errors.New("informe a origem")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	path := filepath.Join(a.cfg.DataDir, "origem.csv")
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, originOutput{}, err
	}
	rows, err := csv.NewReader(strings.NewReader(string(existing))).ReadAll()
	if err != nil {
		return nil, originOutput{}, fmt.Errorf("origem.csv ilegível: %v", err)
	}
	for _, row := range rows {
		if len(row) >= 2 && row[1] == key {
			return nil, originOutput{OK: true, JaRegistrado: true}, nil
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, originOutput{}, err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if len(rows) == 0 {
		if err := w.Write([]string{"data", "telefone", "origem"}); err != nil {
			return nil, originOutput{}, err
		}
	}
	if err := w.Write([]string{a.now().In(saoPaulo).Format("02/01/2006 15:04"), key, origin}); err != nil {
		return nil, originOutput{}, err
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, originOutput{}, err
	}
	return nil, originOutput{OK: true}, nil
}
