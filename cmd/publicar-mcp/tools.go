package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"opdeals/internal/publishq"
)

//go:embed instructions.md
var instructions string

const serverInstructions = "Fila de publicação da Collecta TCG. O Social Collecta usa criar_peca, pedir_aprovacao, situacao, propor_teste e testes_anuncio; o Estrategista só situacao e testes_anuncio. O Publicador Collecta começa toda rodada pela ferramenta inicio, que traz as regras completas: leia inteiras e siga todas. Nada é postado sem aprovação do dono pelo celular."

const (
	captionMax    = 2200
	hashtagMax    = 30
	fileMax       = 250 << 20
	attachMax     = 15 << 20
	scheduleAhead = 14 * 24 * time.Hour
	scheduleSlack = 10 * time.Minute
	carouselMin   = 2
	carouselMax   = 10
	pushCaption   = 900
)

var (
	saoPaulo   = mustLocation("America/Sao_Paulo")
	videoExt   = map[string]bool{".mp4": true, ".mov": true}
	imageExt   = map[string]bool{".jpg": true, ".jpeg": true, ".png": true}
	linkHosts  = map[string][]string{"instagram": {"instagram.com"}, "tiktok": {"tiktok.com"}}
	timeLayout = []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02 15:04"}
)

func mustLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

type app struct {
	cfg      config
	client   *http.Client
	now      func() time.Time
	render   renderFunc
	renderMu sync.Mutex
	feedMu   sync.Mutex
	feed     feedCache
}

func newApp(cfg config, client *http.Client, now func() time.Time) *app {
	return &app{cfg: cfg, client: client, now: now, render: nodeRender(cfg.Node, cfg.MotionDir)}
}

func (a *app) server() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "collecta-publicar", Version: "1.0.0"}, &mcp.ServerOptions{Instructions: serverInstructions})
	mcp.AddTool(s, &mcp.Tool{
		Name:        "pedir_aprovacao",
		Description: "Só para o Social Collecta. Manda uma peça pronta para o dono aprovar pelo celular: canal (instagram ou tiktok), formato (instagram: reels, post ou carrossel; tiktok: video), os arquivos (caminho completo dentro de store/), a legenda final e quando postar. Nada é postado sem o dono tocar em Aprovar.",
	}, a.toolRequest)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "situacao",
		Description: "Para o Social e o Estrategista. Só leitura: as publicações dos últimos 7 dias e as que estão para sair, com a situação de cada uma (esperando o dono, aprovada, rejeitada, publicada com link, falhou com motivo).",
	}, a.toolStatus)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "inicio",
		Description: "Só para o Publicador Collecta. Chame primeiro em toda rodada: devolve a hora em Goiânia, a versão e o texto completo das regras.",
	}, a.toolStart)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "aprovados",
		Description: "Só para o Publicador Collecta. Lista as publicações aprovadas pelo dono que já chegaram no horário, com os arquivos copiados para a pasta de aprovados (os únicos que podem ser enviados no navegador) e a legenda exata.",
	}, a.toolApproved)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "publicado",
		Description: "Só para o Publicador Collecta. Registra que a publicação saiu, com o link do post no Instagram ou no TikTok, e avisa o dono.",
	}, a.toolPublished)
	mcp.AddTool(s, &mcp.Tool{
		Name:        "falhou",
		Description: "Só para o Publicador Collecta. Registra que a publicação não saiu e avisa o dono. permanente = true só quando tentar de novo não resolve (arquivo recusado pela rede, legenda recusada). Falha passageira volta na próxima rodada, até 3 tentativas.",
	}, a.toolFailed)
	a.addPecaTool(s)
	a.addAdTools(s)
	return s
}

type requestInput struct {
	Canal      string   `json:"canal" jsonschema:"instagram ou tiktok"`
	Formato    string   `json:"formato" jsonschema:"instagram: reels, post ou carrossel; tiktok: video"`
	Arquivos   []string `json:"arquivos" jsonschema:"caminhos completos dos arquivos, na ordem do post (ex.: /Users/jjairo/Documents/www/collecta-app/store/motion/out/collecta-tcg-loja-feed.mp4)"`
	Legenda    string   `json:"legenda" jsonschema:"a legenda final, exatamente como vai ao ar, com hashtags"`
	PublicarEm string   `json:"publicar_em" jsonschema:"quando postar, horário de Goiânia: 2026-11-06T15:00"`
	Motivo     string   `json:"motivo,omitempty" jsonschema:"uma linha: por que essa peça, para o dono decidir (ex.: ação de hoje do radar)"`
}

type requestOutput struct {
	ID       string `json:"id"`
	Quando   string `json:"quando"`
	Previa   bool   `json:"previa_no_celular"`
	Proximo  string `json:"proximo"`
	Arquivos int    `json:"arquivos"`
}

func (a *app) toolRequest(ctx context.Context, _ *mcp.CallToolRequest, in requestInput) (*mcp.CallToolResult, requestOutput, error) {
	canal := strings.ToLower(strings.TrimSpace(in.Canal))
	formato := strings.ToLower(strings.TrimSpace(in.Formato))
	if !publishq.ValidFormat(canal, formato) {
		return nil, requestOutput{}, fmt.Errorf("canal %q com formato %q não existe: instagram (reels, post, carrossel) ou tiktok (video)", in.Canal, in.Formato)
	}
	caption, err := checkCaption(in.Legenda)
	if err != nil {
		return nil, requestOutput{}, err
	}
	when, err := a.parseWhen(in.PublicarEm)
	if err != nil {
		return nil, requestOutput{}, err
	}
	files, err := a.checkFiles(formato, in.Arquivos)
	if err != nil {
		return nil, requestOutput{}, err
	}
	var created publishq.Item
	item := publishq.Item{Channel: canal, Format: formato, Files: files, Caption: caption, PostAt: when, Note: cleanLine(in.Motivo)}
	if err := a.call(ctx, http.MethodPost, "/api/publish", item, &created); err != nil {
		return nil, requestOutput{}, err
	}
	preview, err := a.askOwner(ctx, created)
	if err != nil {
		return nil, requestOutput{}, fmt.Errorf("pedido %s criado, mas o aviso não chegou ao dono: %v", created.ID, err)
	}
	return nil, requestOutput{
		ID:       created.ID,
		Quando:   when.In(saoPaulo).Format("02/01 15:04"),
		Previa:   preview,
		Arquivos: len(files),
		Proximo:  "o dono decide pelo celular. Anote o id na seção Peças do seu arquivo do dia e não peça de novo a mesma peça.",
	}, nil
}

func checkCaption(raw string) (string, error) {
	caption := strings.TrimSpace(raw)
	switch {
	case caption == "":
		return "", errors.New("legenda vazia")
	case utf8.RuneCountInString(caption) > captionMax:
		return "", fmt.Errorf("legenda com %d caracteres: o máximo é %d", utf8.RuneCountInString(caption), captionMax)
	case strings.ContainsAny(caption, "—–"):
		return "", errors.New("legenda com travessão: troque por vírgula, dois-pontos ou quebre a frase")
	case strings.Count(caption, "#") > hashtagMax:
		return "", fmt.Errorf("mais de %d hashtags", hashtagMax)
	}
	return caption, nil
}

func (a *app) parseWhen(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	var when time.Time
	var err error
	for _, layout := range timeLayout {
		when, err = time.ParseInLocation(layout, raw, saoPaulo)
		if err == nil {
			break
		}
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("publicar_em inválido %q: use 2026-11-06T15:00 (horário de Goiânia)", raw)
	}
	now := a.now()
	switch {
	case when.Before(now.Add(-scheduleSlack)):
		return time.Time{}, errors.New("publicar_em já passou: escolha um horário a partir de agora")
	case when.After(now.Add(scheduleAhead)):
		return time.Time{}, errors.New("publicar_em passa de 14 dias: peça mais perto da data")
	}
	return when, nil
}

func (a *app) underRoots(path string) bool {
	for _, root := range a.cfg.MediaRoots {
		if real, err := filepath.EvalSymlinks(root); err == nil {
			root = real
		}
		if path == root || strings.HasPrefix(path, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (a *app) checkFiles(formato string, paths []string) ([]publishq.File, error) {
	switch {
	case formato == "carrossel" && (len(paths) < carouselMin || len(paths) > carouselMax):
		return nil, fmt.Errorf("carrossel leva de %d a %d arquivos", carouselMin, carouselMax)
	case formato != "carrossel" && len(paths) != 1:
		return nil, fmt.Errorf("%s leva exatamente 1 arquivo", formato)
	}
	out := make([]publishq.File, 0, len(paths))
	for _, p := range paths {
		if !filepath.IsAbs(p) {
			return nil, fmt.Errorf("caminho %q precisa ser completo", p)
		}
		real, err := filepath.EvalSymlinks(filepath.Clean(p))
		if err != nil {
			return nil, fmt.Errorf("arquivo %s não existe", filepath.Base(p))
		}
		if !a.underRoots(real) {
			return nil, fmt.Errorf("arquivo %s fora das pastas da loja (%s)", filepath.Base(p), strings.Join(a.cfg.MediaRoots, ", "))
		}
		ext := strings.ToLower(filepath.Ext(real))
		switch formato {
		case "reels", "video":
			if !videoExt[ext] {
				return nil, fmt.Errorf("%s precisa de vídeo .mp4 ou .mov, veio %s", formato, filepath.Base(real))
			}
		case "post":
			if !imageExt[ext] {
				return nil, fmt.Errorf("post precisa de imagem .jpg ou .png, veio %s", filepath.Base(real))
			}
		case "carrossel":
			if !imageExt[ext] && !videoExt[ext] {
				return nil, fmt.Errorf("carrossel aceita imagem ou vídeo, veio %s", filepath.Base(real))
			}
		}
		f, err := hashFile(real)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

func hashFile(path string) (publishq.File, error) {
	fh, err := os.Open(path)
	if err != nil {
		return publishq.File{}, fmt.Errorf("não consegui abrir %s: %v", filepath.Base(path), err)
	}
	defer fh.Close()
	info, err := fh.Stat()
	if err != nil {
		return publishq.File{}, err
	}
	if !info.Mode().IsRegular() {
		return publishq.File{}, fmt.Errorf("%s não é um arquivo", filepath.Base(path))
	}
	if info.Size() > fileMax {
		return publishq.File{}, fmt.Errorf("%s tem %d MB: o máximo é %d MB", filepath.Base(path), info.Size()>>20, fileMax>>20)
	}
	sum := sha256.New()
	if _, err := io.Copy(sum, fh); err != nil {
		return publishq.File{}, err
	}
	return publishq.File{Path: path, SHA256: hex.EncodeToString(sum.Sum(nil)), Size: info.Size()}, nil
}

func cleanLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func (a *app) decideURL(it publishq.Item, action string) string {
	q := url.Values{"acao": {action}, "t": {it.Token}}
	return a.cfg.OpdealsURL + "/api/aprovar/" + url.PathEscape(it.ID) + "?" + q.Encode()
}

func (a *app) askOwner(ctx context.Context, it publishq.Item) (bool, error) {
	caption := it.Caption
	if r := []rune(caption); len(r) > pushCaption {
		caption = string(r[:pushCaption]) + "…"
	}
	names := make([]string, 0, len(it.Files))
	for _, f := range it.Files {
		names = append(names, filepath.Base(f.Path))
	}
	body := fmt.Sprintf("%s %s · %s\nArquivos: %s\n", it.Channel, it.Format, it.PostAt.In(saoPaulo).Format("02/01 15:04"), strings.Join(names, ", "))
	if it.Note != "" {
		body += "Por quê: " + it.Note + "\n"
	}
	body += "\n" + caption
	actions := fmt.Sprintf("http, Aprovar, %s, method=POST, clear=true; http, Rejeitar, %s, method=POST, clear=true", a.decideURL(it, "aprovar"), a.decideURL(it, "rejeitar"))
	headers := map[string]string{"Title": "Aprovar post da Collecta", "Actions": actions, "Tags": "camera", "Priority": "high"}
	first := it.Files[0]
	if first.Size > attachMax {
		return false, a.push(ctx, headers, strings.NewReader(body))
	}
	fh, err := os.Open(first.Path)
	if err != nil {
		return false, err
	}
	defer fh.Close()
	headers["Filename"] = filepath.Base(first.Path)
	headers["Message"] = body
	return true, a.push(ctx, headers, fh)
}

func (a *app) push(ctx context.Context, headers map[string]string, body io.Reader) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, a.cfg.NtfyURL, body)
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, encodeHeader(v))
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy respondeu %d", resp.StatusCode)
	}
	return nil
}

func encodeHeader(v string) string {
	v = strings.ReplaceAll(v, "\n", `\n`)
	for _, r := range v {
		if r > 126 {
			return mime.BEncoding.Encode("UTF-8", v)
		}
	}
	return v
}

func (a *app) call(ctx context.Context, method, path string, body, out any) error {
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, a.cfg.OpdealsURL+path, payload)
	if err != nil {
		return err
	}
	req.Header.Set("X-Admin-Token", a.cfg.AdminToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("dashboard fora do ar: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("dashboard respondeu %d em %s: %s", resp.StatusCode, path, strings.TrimSpace(string(msg)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type noInput struct{}

const statusWindow = 7 * 24 * time.Hour

var statusLabel = map[string]string{
	publishq.StatusPending:   "esperando o dono",
	publishq.StatusApproved:  "aprovada",
	publishq.StatusRejected:  "rejeitada",
	publishq.StatusPublished: "publicada",
	publishq.StatusFailed:    "falhou",
}

type statusItem struct {
	ID       string `json:"id"`
	Canal    string `json:"canal"`
	Formato  string `json:"formato"`
	Horario  string `json:"horario"`
	Situacao string `json:"situacao"`
	Legenda  string `json:"inicio_da_legenda"`
	Motivo   string `json:"motivo,omitempty"`
	Link     string `json:"link,omitempty"`
	Erro     string `json:"erro,omitempty"`
}

type statusOutput struct {
	Itens []statusItem `json:"itens"`
}

func (a *app) toolStatus(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, statusOutput, error) {
	var list struct {
		Items []publishq.Item `json:"items"`
	}
	if err := a.call(ctx, http.MethodGet, "/api/publish", nil, &list); err != nil {
		return nil, statusOutput{}, err
	}
	out := statusOutput{Itens: []statusItem{}}
	since := a.now().Add(-statusWindow)
	for _, it := range list.Items {
		if it.PostAt.Before(since) {
			continue
		}
		caption := []rune(it.Caption)
		if len(caption) > 80 {
			caption = append(caption[:80], '…')
		}
		out.Itens = append(out.Itens, statusItem{
			ID: it.ID, Canal: it.Channel, Formato: it.Format, Horario: it.PostAt.In(saoPaulo).Format("02/01 15:04"),
			Situacao: statusLabel[it.Status], Legenda: string(caption), Motivo: it.Note, Link: it.Link, Erro: it.Error,
		})
	}
	return nil, out, nil
}

type startOutput struct {
	Agora  string `json:"agora"`
	Versao string `json:"versao_regras"`
	Regras string `json:"regras"`
}

func (a *app) toolStart(_ context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, startOutput, error) {
	sum := sha256.Sum256([]byte(instructions))
	return nil, startOutput{
		Agora:  a.now().In(saoPaulo).Format("02/01/2006 15:04"),
		Versao: hex.EncodeToString(sum[:])[:8],
		Regras: instructions,
	}, nil
}

type readyItem struct {
	ID         string   `json:"id"`
	Canal      string   `json:"canal"`
	Formato    string   `json:"formato"`
	Arquivos   []string `json:"arquivos"`
	Legenda    string   `json:"legenda"`
	Horario    string   `json:"horario"`
	Tentativas int      `json:"tentativas"`
}

type approvedOutput struct {
	Itens      []readyItem `json:"itens"`
	Recusados  []string    `json:"recusados,omitempty"`
	Agendados  int         `json:"agendados_para_depois"`
	Instrucoes string      `json:"proximo"`
}

func (a *app) stageDir(id string) string {
	return filepath.Join(a.cfg.DataDir, "aprovados", id)
}

func (a *app) stage(it publishq.Item) ([]string, error) {
	dir := a.stageDir(it.ID)
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(it.Files))
	for i, f := range it.Files {
		got, err := hashFile(f.Path)
		if err != nil {
			return nil, err
		}
		if got.SHA256 != f.SHA256 {
			return nil, fmt.Errorf("%s mudou depois da aprovação", filepath.Base(f.Path))
		}
		dst := filepath.Join(dir, fmt.Sprintf("%02d-%s", i+1, filepath.Base(f.Path)))
		if err := copyFile(f.Path, dst); err != nil {
			return nil, err
		}
		out = append(out, dst)
	}
	return out, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func (a *app) toolApproved(ctx context.Context, _ *mcp.CallToolRequest, _ noInput) (*mcp.CallToolResult, approvedOutput, error) {
	var list struct {
		Items []publishq.Item `json:"items"`
	}
	if err := a.call(ctx, http.MethodGet, "/api/publish?status="+publishq.StatusApproved, nil, &list); err != nil {
		return nil, approvedOutput{}, err
	}
	out := approvedOutput{Itens: []readyItem{}, Instrucoes: "publique um por vez, na ordem, e chame publicado ou falhou para cada um"}
	now := a.now()
	for _, it := range list.Items {
		if it.PostAt.After(now) {
			out.Agendados++
			continue
		}
		paths, err := a.stage(it)
		if err != nil {
			msg := "arquivo recusado antes de postar: " + err.Error()
			if failed, ferr := a.fail(ctx, it.ID, msg, true); ferr != nil && failed.ID == "" {
				return nil, approvedOutput{}, ferr
			}
			out.Recusados = append(out.Recusados, it.ID+": "+msg)
			continue
		}
		out.Itens = append(out.Itens, readyItem{
			ID: it.ID, Canal: it.Channel, Formato: it.Format, Arquivos: paths, Legenda: it.Caption,
			Horario: it.PostAt.In(saoPaulo).Format("02/01 15:04"), Tentativas: it.Attempts,
		})
	}
	return nil, out, nil
}

type publishedInput struct {
	ID   string `json:"id" jsonschema:"id que veio de aprovados"`
	Link string `json:"link" jsonschema:"link do post publicado (https://www.instagram.com/p/... ou /reel/..., https://www.tiktok.com/@.../video/...)"`
}

type okOutput struct {
	OK     bool   `json:"ok"`
	Status string `json:"status,omitempty"`
}

func (a *app) channelOf(ctx context.Context, id string) (string, error) {
	var list struct {
		Items []publishq.Item `json:"items"`
	}
	if err := a.call(ctx, http.MethodGet, "/api/publish?status="+publishq.StatusApproved, nil, &list); err != nil {
		return "", err
	}
	for _, it := range list.Items {
		if it.ID == id {
			return it.Channel, nil
		}
	}
	return "", fmt.Errorf("publicação %s não está aprovada: confira o id em aprovados", id)
}

func validLink(channel, raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.Path == "" || u.Path == "/" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, h := range linkHosts[channel] {
		if host == h || strings.HasSuffix(host, "."+h) {
			return true
		}
	}
	return false
}

func (a *app) toolPublished(ctx context.Context, _ *mcp.CallToolRequest, in publishedInput) (*mcp.CallToolResult, okOutput, error) {
	channel, err := a.channelOf(ctx, in.ID)
	if err != nil {
		return nil, okOutput{}, err
	}
	if !validLink(channel, in.Link) {
		return nil, okOutput{}, fmt.Errorf("link %q não é de um post do %s: abra o post publicado e copie o endereço", in.Link, channel)
	}
	var done publishq.Item
	if err := a.call(ctx, http.MethodPost, "/api/publish/"+url.PathEscape(in.ID)+"/published", map[string]string{"link": strings.TrimSpace(in.Link)}, &done); err != nil {
		return nil, okOutput{}, err
	}
	_ = os.RemoveAll(a.stageDir(in.ID))
	a.logLine(fmt.Sprintf("publicado %s %s %s %s", in.ID, done.Channel, done.Format, done.Link))
	msg := fmt.Sprintf("%s %s no ar: %s", done.Channel, done.Format, done.Link)
	if err := a.push(ctx, map[string]string{"Title": "Post publicado", "Click": done.Link, "Tags": "white_check_mark"}, strings.NewReader(msg)); err != nil {
		return nil, okOutput{OK: true, Status: done.Status}, fmt.Errorf("post registrado, mas o aviso não chegou ao dono: %v", err)
	}
	return nil, okOutput{OK: true, Status: done.Status}, nil
}

type failedInput struct {
	ID         string `json:"id" jsonschema:"id que veio de aprovados"`
	Motivo     string `json:"motivo" jsonschema:"o que aconteceu, em uma linha"`
	Permanente bool   `json:"permanente,omitempty" jsonschema:"true só quando tentar de novo não resolve"`
}

func (a *app) fail(ctx context.Context, id, motivo string, permanent bool) (publishq.Item, error) {
	var it publishq.Item
	body := map[string]any{"error": motivo, "permanent": permanent}
	if err := a.call(ctx, http.MethodPost, "/api/publish/"+url.PathEscape(id)+"/failed", body, &it); err != nil {
		return publishq.Item{}, err
	}
	a.logLine(fmt.Sprintf("falhou %s tentativa %d status %s: %s", id, it.Attempts, it.Status, motivo))
	priority, title := "default", "Post não saiu, tento de novo"
	if it.Status == publishq.StatusFailed {
		_ = os.RemoveAll(a.stageDir(id))
		priority, title = "high", "Post não saiu"
	}
	text := fmt.Sprintf("%s %s (%s): %s", it.Channel, it.Format, it.PostAt.In(saoPaulo).Format("02/01 15:04"), motivo)
	if err := a.push(ctx, map[string]string{"Title": title, "Priority": priority, "Tags": "warning"}, strings.NewReader(text)); err != nil {
		return it, fmt.Errorf("falha registrada, mas o aviso não chegou ao dono: %v", err)
	}
	return it, nil
}

func (a *app) toolFailed(ctx context.Context, _ *mcp.CallToolRequest, in failedInput) (*mcp.CallToolResult, okOutput, error) {
	motivo := cleanLine(in.Motivo)
	if strings.TrimSpace(in.ID) == "" || motivo == "" {
		return nil, okOutput{}, errors.New("informe o id e o motivo")
	}
	it, err := a.fail(ctx, in.ID, motivo, in.Permanente)
	if err != nil {
		return nil, okOutput{}, err
	}
	return nil, okOutput{OK: true, Status: it.Status}, nil
}

func (a *app) logLine(line string) {
	f, err := os.OpenFile(filepath.Join(a.cfg.DataDir, "publicacoes.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", a.now().In(saoPaulo).Format("2006-01-02 15:04"), line)
}
