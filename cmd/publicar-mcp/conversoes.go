package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"opdeals/internal/metaevents"
)

// Compras para o Meta. Duas partes, as duas só com dado do cliente já com hash,
// montado pelo painel (GET /api/meta/events e /api/meta/customers):
//   - a cada rodada de anúncios, com META_CAPI=on, as compras dos últimos 7 dias
//     (site, Liga e balcão) vão ao Pixel como Purchase pela Conversions API;
//   - uma vez, com "publicar-mcp clientes-meta", quem comprou nos últimos 180 dias
//     vai para a lista de clientes do Meta (META_LISTA_CLIENTES).
// Nada de cliente é gravado em arquivo ou log: o arquivo de estado guarda só os
// ids dos eventos já enviados, e o log só contagens e ids.

const (
	capiStateFile   = "capi-enviados.json"
	capiKeep        = 10 * 24 * time.Hour
	customersBatch  = 10000
	customersDays   = 180
	conversionsDays = 7
)

type capiEvents struct {
	Events  []metaevents.Event `json:"events"`
	Skipped struct {
		SiteSemNavegador int `json:"site_sem_navegador"`
		LigaSemTelefone  int `json:"liga_sem_telefone"`
		BalcaoSemContato int `json:"balcao_sem_contato"`
	} `json:"skipped"`
}

type capiResult struct {
	Sent, Already, Failed int
}

// sendConversions manda ao Pixel as compras ainda não enviadas. Com testCode
// (META_CAPI_TESTE), os eventos vão para "Testar eventos" do Gerenciador e não
// contam como enviados, para a rodada de verdade mandar de novo.
func sendConversions(ctx context.Context, a *app, g *graph, testCode string) (capiResult, error) {
	var res capiResult
	if g.cfg.PixelID == "" {
		return res, errors.New("META_PIXEL não definido em anuncios.env: as compras não têm para onde ir")
	}
	var in capiEvents
	if err := a.call(ctx, http.MethodGet, fmt.Sprintf("/api/meta/events?days=%d", conversionsDays), nil, &in); err != nil {
		return res, fmt.Errorf("não li as compras do painel: %w", err)
	}
	statePath := filepath.Join(a.cfg.DataDir, capiStateFile)
	sent := loadSent(statePath)
	now := a.now()
	var errs []error
	for _, ev := range in.Events {
		if _, done := sent[ev.EventID]; done {
			res.Already++
			continue
		}
		params := url.Values{"data": {jsonParam([]metaevents.Event{ev})}}
		if testCode != "" {
			params.Set("test_event_code", testCode)
		}
		var out struct {
			EventsReceived int `json:"events_received"`
		}
		if err := g.post(ctx, g.cfg.PixelID+"/events", params, &out); err != nil {
			res.Failed++
			errs = append(errs, fmt.Errorf("compra %s: %v", ev.EventID, err))
			continue
		}
		res.Sent++
		if testCode == "" {
			sent[ev.EventID] = now
		}
	}
	for id, at := range sent {
		if now.Sub(at) > capiKeep {
			delete(sent, id)
		}
	}
	if err := saveSent(statePath, sent); err != nil {
		errs = append(errs, err)
	}
	mode := ""
	if testCode != "" {
		mode = " (teste)"
	}
	a.adLog(fmt.Sprintf("conversões%s: %d enviadas, %d já enviadas, %d falharam; fora: %d do site sem navegador, %d da Liga sem telefone, %d do balcão sem contato",
		mode, res.Sent, res.Already, res.Failed, in.Skipped.SiteSemNavegador, in.Skipped.LigaSemTelefone, in.Skipped.BalcaoSemContato))
	for _, err := range errs {
		a.adLog("conversão não enviada: " + err.Error())
	}
	return res, errors.Join(errs...)
}

func loadSent(path string) map[string]time.Time {
	sent := map[string]time.Time{}
	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &sent)
	}
	return sent
}

func saveSent(path string, sent map[string]time.Time) error {
	data, err := json.MarshalIndent(sent, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

type customersIn struct {
	Schema []string   `json:"schema"`
	Data   [][]string `json:"data"`
	Counts struct {
		Site   int `json:"site"`
		Liga   int `json:"liga"`
		Balcao int `json:"balcao"`
	} `json:"counts"`
}

type customersResult struct {
	People, Site, Liga, Balcao int
	Received, Invalid          int
}

// uploadCustomers manda quem comprou nos últimos dias para a lista de clientes
// do Meta (Custom Audience de lista), em lotes. Repetir não duplica: o Meta casa
// as pessoas pelo hash.
func uploadCustomers(ctx context.Context, a *app, g *graph, audience string, days int) (customersResult, error) {
	var res customersResult
	if audience == "" {
		return res, errors.New("META_LISTA_CLIENTES não definido em anuncios.env: crie a lista de clientes no Gerenciador de Anúncios e ponha o id lá")
	}
	var in customersIn
	if err := a.call(ctx, http.MethodGet, fmt.Sprintf("/api/meta/customers?days=%d", days), nil, &in); err != nil {
		return res, fmt.Errorf("não li os clientes do painel: %w", err)
	}
	res.People, res.Site, res.Liga, res.Balcao = len(in.Data), in.Counts.Site, in.Counts.Liga, in.Counts.Balcao
	for start := 0; start < len(in.Data); start += customersBatch {
		end := min(start+customersBatch, len(in.Data))
		payload := map[string]any{"schema": in.Schema, "data": in.Data[start:end]}
		var out struct {
			Received int `json:"num_received"`
			Invalid  int `json:"num_invalid_entries"`
		}
		if err := g.post(ctx, audience+"/users", url.Values{"payload": {jsonParam(payload)}}, &out); err != nil {
			return res, fmt.Errorf("lote %d a %d da lista de clientes: %v", start+1, end, err)
		}
		res.Received += out.Received
		res.Invalid += out.Invalid
	}
	return res, nil
}

// runCustomers é o "publicar-mcp clientes-meta": a carga do histórico, uma vez.
func runCustomers() error {
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
	res, err := uploadCustomers(ctx, a, &graph{cfg: mc, client: client}, getenv("META_LISTA_CLIENTES"), customersDays)
	fmt.Printf("lista de clientes: %d pessoas dos últimos %d dias (site %d, Liga %d, balcão %d); o Meta recebeu %d, %d inválidas\n",
		res.People, customersDays, res.Site, res.Liga, res.Balcao, res.Received, res.Invalid)
	a.adLog(fmt.Sprintf("lista de clientes: %d pessoas enviadas, %d recebidas, %d inválidas", res.People, res.Received, res.Invalid))
	return err
}
