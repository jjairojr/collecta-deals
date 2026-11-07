package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"opdeals/internal/adtestq"
)

const (
	adTextMax     = 400
	adTitleMax    = 40
	adHookMax     = 40
	adStatusSince = 14 * 24 * time.Hour
)

var adStatusLabel = map[string]string{
	adtestq.StatusPending:  "esperando o dono",
	adtestq.StatusApproved: "aprovado, sobe na próxima rodada",
	adtestq.StatusRejected: "rejeitado",
	adtestq.StatusRunning:  "no ar",
	adtestq.StatusDone:     "terminado",
	adtestq.StatusFailed:   "falhou",
}

func (a *app) addAdTools(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "propor_teste",
		Description: "Só para o Social Collecta. Propõe um teste de anúncio no Meta (Instagram e Facebook) com 2 a 5 criativos diferentes, para o dono aprovar pelo celular. Depois do Aprovar, o teste sobe sozinho com a verba total aprovada e para na data de fim; os criativos que perdem são pausados sozinhos. Teto de R$ 150 por semana somando todos os testes.",
	}, a.toolProposeAdTest)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "testes_anuncio",
		Description: "Para o Social e o Estrategista. Só leitura: os testes de anúncio dos últimos 14 dias, a situação de cada um, o que foi pausado e por quê, o resumo com o vencedor dos que terminaram e a verba que ainda cabe na semana.",
	}, a.toolAdTests)
}

type adCreativeInput struct {
	Arquivo string `json:"arquivo" jsonschema:"caminho completo do vídeo (.mp4 gerado pelo criar_peca, que tem a prévia ao lado) ou da imagem (.png ou .jpg)"`
	Texto   string `json:"texto" jsonschema:"texto principal do anúncio, até 400 letras, sem travessão, com os preços do site"`
	Titulo  string `json:"titulo,omitempty" jsonschema:"título curto, até 40 letras (aparece embaixo da imagem no Facebook)"`
	Gancho  string `json:"gancho" jsonschema:"o que este criativo testa, até 40 letras (ex.: preço no Pix, contagem, carta rara)"`
}

type proposeInput struct {
	Objetivo   string            `json:"objetivo" jsonschema:"seguidores (leva para o perfil do Instagram) ou vendas (otimiza compra no site)"`
	Publico    string            `json:"publico" jsonschema:"goiania (30 km da loja, 18 a 45 anos) ou brasil (Brasil, interesse em Pokémon, One Piece e card game)"`
	Link       string            `json:"link,omitempty" jsonschema:"só em vendas: página do site para onde o anúncio leva (https://www.collectatcg.com.br/...)"`
	Criativos  []adCreativeInput `json:"criativos" jsonschema:"de 2 a 5 criativos, cada um com um gancho diferente"`
	VerbaReais int               `json:"verba_reais" jsonschema:"verba total do teste em reais, de 30 a 150, dentro do que cabe na semana"`
	Dias       int               `json:"dias" jsonschema:"duração, de 3 a 7 dias"`
	Motivo     string            `json:"motivo" jsonschema:"uma linha para o dono decidir: o que o teste quer descobrir"`
}

type proposeOutput struct {
	ID          string `json:"id"`
	Verba       string `json:"verba"`
	SobraSemana string `json:"sobra_na_semana"`
	Proximo     string `json:"proximo"`
}

func reais(cents int64) string {
	return fmt.Sprintf("R$ %d,%02d", cents/100, cents%100)
}

func (a *app) toolProposeAdTest(ctx context.Context, _ *mcp.CallToolRequest, in proposeInput) (*mcp.CallToolResult, proposeOutput, error) {
	t := adtestq.Test{
		Objective:   strings.ToLower(strings.TrimSpace(in.Objetivo)),
		Audience:    strings.ToLower(strings.TrimSpace(in.Publico)),
		BudgetCents: int64(in.VerbaReais) * 100,
		Days:        in.Dias,
		Note:        cleanLine(in.Motivo),
	}
	switch {
	case t.Objective != "seguidores" && t.Objective != "vendas":
		return nil, proposeOutput{}, fmt.Errorf("objetivo %q não existe: seguidores ou vendas", in.Objetivo)
	case t.Audience != "goiania" && t.Audience != "brasil":
		return nil, proposeOutput{}, fmt.Errorf("publico %q não existe: goiania ou brasil", in.Publico)
	case t.BudgetCents < adtestq.MinBudgetCents || t.BudgetCents > adtestq.WeeklyCapCents:
		return nil, proposeOutput{}, fmt.Errorf("verba_reais vai de %d a %d", adtestq.MinBudgetCents/100, adtestq.WeeklyCapCents/100)
	case t.Days < adtestq.MinDays || t.Days > adtestq.MaxDays:
		return nil, proposeOutput{}, fmt.Errorf("dias vai de %d a %d", adtestq.MinDays, adtestq.MaxDays)
	case len(in.Criativos) < adtestq.MinCreatives || len(in.Criativos) > adtestq.MaxCreatives:
		return nil, proposeOutput{}, fmt.Errorf("de %d a %d criativos", adtestq.MinCreatives, adtestq.MaxCreatives)
	case t.Note == "":
		return nil, proposeOutput{}, errors.New("motivo vazio")
	}
	if t.Objective == "vendas" {
		link, err := siteLink(in.Link)
		if err != nil {
			return nil, proposeOutput{}, err
		}
		t.Link = link
	}
	available, err := a.adBudgetLeft(ctx)
	if err != nil {
		return nil, proposeOutput{}, err
	}
	if t.BudgetCents > available {
		return nil, proposeOutput{}, fmt.Errorf("cabem só %s na semana: proponha um teste menor ou espere", reais(available))
	}
	stage := filepath.Join(a.cfg.DataDir, "anuncios", randomID())
	creatives, err := a.stageCreatives(stage, in.Criativos)
	if err != nil {
		_ = os.RemoveAll(stage)
		return nil, proposeOutput{}, err
	}
	t.Creatives = creatives
	var created adtestq.Test
	if err := a.call(ctx, http.MethodPost, "/api/adtests", t, &created); err != nil {
		_ = os.RemoveAll(stage)
		return nil, proposeOutput{}, err
	}
	a.adLog(fmt.Sprintf("proposto %s %s %s %s %d dias %d criativos", created.ID, t.Objective, t.Audience, reais(t.BudgetCents), t.Days, len(creatives)))
	if err := a.askAdApproval(ctx, created, available-t.BudgetCents); err != nil {
		return nil, proposeOutput{}, fmt.Errorf("teste %s criado, mas o aviso não chegou ao dono: %v", created.ID, err)
	}
	return nil, proposeOutput{
		ID:          created.ID,
		Verba:       fmt.Sprintf("%s em %d dias", reais(t.BudgetCents), t.Days),
		SobraSemana: reais(available - t.BudgetCents),
		Proximo:     "o dono decide pelo celular. Anote o id na seção Meta Ads do seu arquivo do dia e acompanhe com testes_anuncio.",
	}, nil
}

func siteLink(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || (u.Host != siteHost && u.Host != strings.TrimPrefix(siteHost, "www.")) {
		return "", fmt.Errorf("link %q precisa ser uma página de https://%s", raw, siteHost)
	}
	u.Host = siteHost
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func (a *app) adBudgetLeft(ctx context.Context) (int64, error) {
	var list struct {
		AvailableCents int64 `json:"availableCents"`
	}
	if err := a.call(ctx, http.MethodGet, "/api/adtests?status="+adtestq.StatusRunning, nil, &list); err != nil {
		return 0, err
	}
	return list.AvailableCents, nil
}

func (a *app) stageCreatives(stage string, in []adCreativeInput) ([]adtestq.Creative, error) {
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return nil, err
	}
	out := make([]adtestq.Creative, 0, len(in))
	for i, c := range in {
		text := strings.TrimSpace(c.Texto)
		title := cleanLine(c.Titulo)
		hook := cleanLine(c.Gancho)
		limits := []textLimit{{"texto", text, 1, adTextMax}, {"gancho", hook, 1, adHookMax}}
		if title != "" {
			limits = append(limits, textLimit{"titulo", title, 1, adTitleMax})
		}
		for _, l := range limits {
			if err := l.check(); err != nil {
				return nil, fmt.Errorf("criativo %d: %v", i+1, err)
			}
		}
		if !filepath.IsAbs(c.Arquivo) {
			return nil, fmt.Errorf("criativo %d: caminho %q precisa ser completo", i+1, c.Arquivo)
		}
		real, err := filepath.EvalSymlinks(filepath.Clean(c.Arquivo))
		if err != nil || !a.underRoots(real) {
			return nil, fmt.Errorf("criativo %d: arquivo %s não existe ou está fora das pastas da loja", i+1, filepath.Base(c.Arquivo))
		}
		ext := strings.ToLower(filepath.Ext(real))
		kind := "imagem"
		thumb := ""
		switch {
		case ext == ".mp4":
			kind = "video"
			thumb = strings.TrimSuffix(real, ext) + "-previa-1.png"
			if _, err := os.Stat(thumb); err != nil {
				return nil, fmt.Errorf("criativo %d: o vídeo precisa ser um do criar_peca (com a prévia ao lado)", i+1)
			}
		case imageExt[ext]:
		default:
			return nil, fmt.Errorf("criativo %d: use vídeo .mp4 ou imagem .png ou .jpg", i+1)
		}
		f, err := hashFile(real)
		if err != nil {
			return nil, err
		}
		dst := filepath.Join(stage, fmt.Sprintf("%02d%s", i+1, ext))
		if err := copyFile(real, dst); err != nil {
			return nil, err
		}
		cr := adtestq.Creative{Path: dst, SHA256: f.SHA256, Size: f.Size, Kind: kind, Text: text, Headline: title, Hook: hook}
		if thumb != "" {
			cr.Thumb = filepath.Join(stage, fmt.Sprintf("%02d-capa.png", i+1))
			if err := copyFile(thumb, cr.Thumb); err != nil {
				return nil, err
			}
		}
		out = append(out, cr)
	}
	return out, nil
}

var objectiveLabel = map[string]string{"seguidores": "seguidores (perfil do Instagram)", "vendas": "vendas no site"}
var audienceLabel = map[string]string{"goiania": "Goiânia, 30 km", "brasil": "Brasil, interesse em TCG"}

func (a *app) adDecideURL(t adtestq.Test, action string) string {
	q := url.Values{"acao": {action}, "t": {t.Token}}
	return a.cfg.OpdealsURL + "/api/aprovar-anuncio/" + url.PathEscape(t.ID) + "?" + q.Encode()
}

func (a *app) askAdApproval(ctx context.Context, t adtestq.Test, left int64) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Objetivo: %s\nPúblico: %s\nVerba: %s em %d dias (sobra %s na semana)\n", objectiveLabel[t.Objective], audienceLabel[t.Audience], reais(t.BudgetCents), t.Days, reais(left))
	if t.Link != "" {
		fmt.Fprintf(&b, "Leva para: %s\n", t.Link)
	}
	fmt.Fprintf(&b, "Por quê: %s\n", t.Note)
	for i, c := range t.Creatives {
		text := []rune(c.Text)
		if len(text) > 160 {
			text = append(text[:160], '…')
		}
		fmt.Fprintf(&b, "\n%d. %s (%s): %s", i+1, c.Hook, c.Kind, string(text))
	}
	actions := fmt.Sprintf("http, Aprovar, %s, method=POST, clear=true; http, Rejeitar, %s, method=POST, clear=true", a.adDecideURL(t, "aprovar"), a.adDecideURL(t, "rejeitar"))
	headers := map[string]string{"Title": "Aprovar teste de anúncio", "Actions": actions, "Tags": "moneybag", "Priority": "high"}
	preview := t.Creatives[0].Thumb
	if preview == "" {
		preview = t.Creatives[0].Path
	}
	info, err := os.Stat(preview)
	if err != nil || info.Size() > attachMax {
		return a.push(ctx, headers, strings.NewReader(b.String()))
	}
	fh, err := os.Open(preview)
	if err != nil {
		return err
	}
	defer fh.Close()
	headers["Filename"] = filepath.Base(preview)
	headers["Message"] = b.String()
	return a.push(ctx, headers, fh)
}

type adStatusAd struct {
	Gancho  string `json:"gancho"`
	Pausado bool   `json:"pausado,omitempty"`
	Motivo  string `json:"motivo,omitempty"`
}

type adStatusItem struct {
	ID       string       `json:"id"`
	Objetivo string       `json:"objetivo"`
	Publico  string       `json:"publico"`
	Verba    string       `json:"verba"`
	Situacao string       `json:"situacao"`
	Motivo   string       `json:"motivo,omitempty"`
	Termina  string       `json:"termina,omitempty"`
	Anuncios []adStatusAd `json:"anuncios,omitempty"`
	Resumo   string       `json:"resumo,omitempty"`
	Erro     string       `json:"erro,omitempty"`
}

type adStatusOutput struct {
	Testes      []adStatusItem `json:"testes"`
	SobraSemana string         `json:"sobra_na_semana"`
}

func (a *app) toolAdTests(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, adStatusOutput, error) {
	var list struct {
		Items          []adtestq.Test `json:"items"`
		AvailableCents int64          `json:"availableCents"`
	}
	if err := a.call(ctx, http.MethodGet, "/api/adtests", nil, &list); err != nil {
		return nil, adStatusOutput{}, err
	}
	out := adStatusOutput{Testes: []adStatusItem{}, SobraSemana: reais(list.AvailableCents)}
	since := a.now().Add(-adStatusSince)
	for _, t := range list.Items {
		if t.CreatedAt.Before(since) {
			continue
		}
		item := adStatusItem{
			ID: t.ID, Objetivo: t.Objective, Publico: t.Audience, Verba: fmt.Sprintf("%s em %d dias", reais(t.BudgetCents), t.Days),
			Situacao: adStatusLabel[t.Status], Motivo: t.Note, Resumo: t.Summary, Erro: t.Error,
		}
		if !t.EndsAt.IsZero() {
			item.Termina = t.EndsAt.In(saoPaulo).Format("02/01 15:04")
		}
		if t.Meta != nil {
			for _, ad := range t.Meta.Ads {
				hook := ""
				if ad.Creative >= 0 && ad.Creative < len(t.Creatives) {
					hook = t.Creatives[ad.Creative].Hook
				}
				item.Anuncios = append(item.Anuncios, adStatusAd{Gancho: hook, Pausado: ad.Paused, Motivo: ad.Reason})
			}
		}
		out.Testes = append(out.Testes, item)
	}
	return nil, out, nil
}

func (a *app) adLog(line string) {
	f, err := os.OpenFile(filepath.Join(a.cfg.DataDir, "anuncios.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", a.now().In(saoPaulo).Format("2006-01-02 15:04"), line)
}
