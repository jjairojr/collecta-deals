package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"opdeals/internal/whatsapp"
)

const (
	modeDraft = "rascunho"
	modeTest  = "teste"
	modeLive  = "ao_vivo"
)

type config struct {
	OpdealsURL string
	SiteURL    string
	NtfyURL    string
	Mode       string
	TestPhones map[string]bool
	DataDir    string
	AgentToken string
	NewsPath   string
}

func loadConfig(getenv func(string) string) (config, error) {
	cfg := config{
		OpdealsURL: strings.TrimRight(cmp.Or(getenv("OPDEALS_URL"), "https://collecta-deals-production.up.railway.app"), "/"),
		SiteURL:    strings.TrimRight(cmp.Or(getenv("SITE_URL"), "https://www.collectatcg.com.br"), "/"),
		Mode:       cmp.Or(getenv("ATENDENTE_MODO"), modeDraft),
		TestPhones: map[string]bool{},
		DataDir:    getenv("ATENDENTE_DADOS"),
		AgentToken: getenv("POSTSALE_AGENT_TOKEN"),
		NewsPath:   getenv("ATENDENTE_NOVIDADES"),
	}
	switch {
	case getenv("NTFY_URL") != "":
		cfg.NtfyURL = getenv("NTFY_URL")
	case getenv("NTFY_TOPIC") != "":
		cfg.NtfyURL = "https://ntfy.sh/" + getenv("NTFY_TOPIC")
	default:
		return cfg, errors.New("NTFY_TOPIC não definido: sem ele o atendente não consegue chamar o dono")
	}
	switch cfg.Mode {
	case modeDraft, modeTest, modeLive:
	default:
		return cfg, fmt.Errorf("ATENDENTE_MODO inválido: %q (use rascunho, teste ou ao_vivo)", cfg.Mode)
	}
	for _, p := range strings.Split(getenv("ATENDENTE_TESTE_TELEFONES"), ",") {
		if key := whatsapp.PhoneKey(p); len(key) >= 10 {
			cfg.TestPhones[key] = true
		}
	}
	if cfg.Mode == modeTest && len(cfg.TestPhones) == 0 {
		return cfg, errors.New("modo teste precisa de ATENDENTE_TESTE_TELEFONES")
	}
	if cfg.DataDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return cfg, err
		}
		cfg.DataDir = filepath.Join(home, ".collecta-atendente")
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
	if cfg.NewsPath == "" {
		exe, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		cfg.NewsPath = filepath.Join(filepath.Dir(exe), "novidades.md")
	}
	a := newApp(cfg, &http.Client{Timeout: 15 * time.Second}, time.Now)
	if err := a.server().Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
