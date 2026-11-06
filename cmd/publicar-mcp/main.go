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
	MediaRoots []string
}

func loadConfig(getenv func(string) string) (config, error) {
	cfg := config{
		OpdealsURL: strings.TrimRight(cmp.Or(getenv("OPDEALS_URL"), "https://collecta-deals-production.up.railway.app"), "/"),
		AdminToken: getenv("OPDEALS_ADMIN_TOKEN"),
		DataDir:    cmp.Or(getenv("PUBLICAR_DADOS"), "/Users/Shared/collecta-bot/publicar"),
	}
	if cfg.AdminToken == "" {
		return cfg, errors.New("OPDEALS_ADMIN_TOKEN não definido: sem ele não há fila de publicação")
	}
	switch {
	case getenv("NTFY_URL") != "":
		cfg.NtfyURL = getenv("NTFY_URL")
	case getenv("NTFY_TOPIC") != "":
		cfg.NtfyURL = "https://ntfy.sh/" + getenv("NTFY_TOPIC")
	default:
		return cfg, errors.New("NTFY_TOPIC não definido: sem ele o pedido de aprovação não chega ao dono")
	}
	roots := cmp.Or(getenv("PUBLICAR_MIDIA"), "/Users/jjairo/Documents/www/collecta-app/store")
	for _, r := range strings.Split(roots, ":") {
		if r = strings.TrimSpace(r); r != "" {
			cfg.MediaRoots = append(cfg.MediaRoots, filepath.Clean(r))
		}
	}
	return cfg, nil
}

func main() {
	cfg, err := loadConfig(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cfg.DataDir, "aprovados"), 0o755); err != nil {
		log.Fatal(err)
	}
	a := newApp(cfg, &http.Client{Timeout: 2 * time.Minute}, time.Now)
	if err := a.server().Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
