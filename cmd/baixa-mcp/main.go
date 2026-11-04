package main

import (
	"cmp"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type config struct {
	OpdealsURL string
	AdminToken string
	NtfyURL    string
	DataDir    string
}

func loadConfig(getenv func(string) string) (config, error) {
	cfg := config{
		OpdealsURL: strings.TrimRight(cmp.Or(getenv("OPDEALS_URL"), "https://collecta-deals-production.up.railway.app"), "/"),
		AdminToken: getenv("OPDEALS_ADMIN_TOKEN"),
		DataDir:    getenv("BAIXA_DADOS"),
	}
	if cfg.AdminToken == "" {
		return cfg, errors.New("OPDEALS_ADMIN_TOKEN não definido: sem ele não há como dar baixa no dashboard")
	}
	switch {
	case getenv("NTFY_URL") != "":
		cfg.NtfyURL = getenv("NTFY_URL")
	case getenv("NTFY_TOPIC") != "":
		cfg.NtfyURL = "https://ntfy.sh/" + getenv("NTFY_TOPIC")
	default:
		return cfg, errors.New("NTFY_TOPIC não definido: sem ele o relatório não chega ao dono")
	}
	if cfg.DataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return cfg, err
		}
		cfg.DataDir = filepath.Join(home, ".collecta-baixa")
	}
	return cfg, nil
}

func main() {
	cfg, err := loadConfig(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		log.Fatal(err)
	}
	a := newApp(cfg, &http.Client{Timeout: 30 * time.Second}, time.Now)
	if err := a.server().Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
