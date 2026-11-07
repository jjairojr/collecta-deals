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
)

type config struct {
	OpdealsURL string
	AdminToken string
	NtfyURL    string
	DataDir    string
	MediaRoots []string
	MotionDir  string
	FeedURL    string
	Node       string
	TCGCSVURL  string
	LigaDir    string
	AdsEnv     string
}

func loadConfig(getenv func(string) string) (config, error) {
	cfg := config{
		OpdealsURL: strings.TrimRight(cmp.Or(getenv("OPDEALS_URL"), "https://collecta-deals-production.up.railway.app"), "/"),
		AdminToken: getenv("OPDEALS_ADMIN_TOKEN"),
		DataDir:    cmp.Or(getenv("PUBLICAR_DADOS"), "/Users/Shared/collecta-bot/publicar"),
		MotionDir:  filepath.Clean(cmp.Or(getenv("PUBLICAR_MOTION"), "/Users/jjairo/Documents/www/collecta-app/store/motion")),
		FeedURL:    cmp.Or(getenv("PUBLICAR_CATALOGO"), "https://www.collectatcg.com.br/feed/meta.csv"),
		Node:       cmp.Or(getenv("PUBLICAR_NODE"), "/opt/homebrew/bin/node"),
		TCGCSVURL:  strings.TrimRight(cmp.Or(getenv("PUBLICAR_TCGCSV"), "https://tcgcsv.com"), "/"),
		LigaDir:    filepath.Clean(cmp.Or(getenv("PUBLICAR_LIGA_DADOS"), "/Users/jjairo/Documents/www/opdeals/data")),
		AdsEnv:     cmp.Or(getenv("PUBLICAR_ANUNCIOS_ENV"), defaultAdsEnv),
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

const (
	adsRunTimeout = 25 * time.Minute
	defaultAdsEnv = "/Users/Shared/collecta-bot/publicar/anuncios.env"
)

func runAds() error {
	envPath := cmp.Or(os.Getenv("PUBLICAR_ANUNCIOS_ENV"), defaultAdsEnv)
	file, err := readEnvFile(envPath)
	if err != nil {
		return fmt.Errorf("não consegui ler %s: %w", envPath, err)
	}
	getenv := func(k string) string {
		if v, ok := file[k]; ok {
			return v
		}
		return os.Getenv(k)
	}
	cfg, err := loadConfig(getenv)
	if err != nil {
		return err
	}
	mc, err := loadMetaConfig(getenv)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	a := newApp(cfg, client, time.Now)
	ctx, cancel := context.WithTimeout(context.Background(), adsRunTimeout)
	defer cancel()
	return newExecutor(a, mc, client).run(ctx)
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "anuncios" {
		if err := runAds(); err != nil {
			log.Fatal(err)
		}
		return
	}
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
