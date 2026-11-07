package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	feedTTL        = 10 * time.Minute
	feedMax        = 20 << 20
	photoMax       = 15 << 20
	renderTimeout  = 8 * time.Minute
	rendersPerDay  = 30
	productsMax    = 3
	stagingKeep    = 3 * 24 * time.Hour
	countdownLimit = 30
	siteHost       = "www.collectatcg.com.br"
)

var (
	inauguration  = time.Date(2026, 11, 15, 0, 0, 0, 0, saoPaulo)
	productPath   = regexp.MustCompile(`^/(carta|selado|acessorio)/[a-z0-9-]+$`)
	installmentRe = regexp.MustCompile(`em até (\d+)x`)
	localImage    = regexp.MustCompile(`^(products|cards)/[A-Za-z0-9._-]+\.(png|jpg|jpeg|webp)$`)
	slugWord      = regexp.MustCompile(`[^a-z0-9]+`)
	pecaModels    = map[string]bool{"produto": true, "contagem": true, "carta": true}
	pecaKinds     = map[string]string{"video": ".mp4", "arte": ".png"}
	photoTypes    = map[string]string{"image/jpeg": ".jpg", "image/png": ".png"}
)

type textLimit struct {
	field string
	value string
	min   int
	max   int
}

type pecaProdutoInput struct {
	Link   string `json:"link" jsonschema:"link do produto no site: https://www.collectatcg.com.br/selado/... ou /carta/... ou /acessorio/..."`
	Nome   string `json:"nome,omitempty" jsonschema:"nome curto que aparece no vídeo, até 28 letras (ex.: Booster Box Reinado Delta). Sem ele vale o nome do site, se couber"`
	Imagem string `json:"imagem,omitempty" jsonschema:"opcional: recorte pronto da pasta store/motion/public, como products/reinado-box.png ou cards/card-op-luffy-manga.webp. Sem ele vale a foto do site"`
}

type pecaFinalInput struct {
	Titulo   string   `json:"titulo" jsonschema:"título do cartão final, até 24 letras (ex.: GARANTA O SEU!)"`
	Destaque []string `json:"destaque,omitempty" jsonschema:"palavras do título pintadas de verde"`
	Oferta   string   `json:"oferta" jsonschema:"selo verde do cartão final, até 28 letras (ex.: Envio pra todo o Brasil)"`
	Linhas   []string `json:"linhas" jsonschema:"de 1 a 3 linhas curtas, até 32 letras cada (ex.: collectatcg.com.br, Loja física em Goiânia)"`
	CTA      string   `json:"cta" jsonschema:"botão final, até 18 letras (ex.: Link na bio)"`
}

type pecaInput struct {
	Modelo   string             `json:"modelo" jsonschema:"produto (1 a 3 produtos com preço), carta (1 carta avulsa com preço) ou contagem (faltam N dias para a inauguração, com até 3 produtos)"`
	Formato  string             `json:"formato" jsonschema:"video (Reels e TikTok, 1080x1920) ou arte (post do Instagram, 1080x1350)"`
	Etiqueta string             `json:"etiqueta" jsonschema:"linha pequena no topo, até 32 letras (ex.: Pokémon · Selado)"`
	Gancho   string             `json:"gancho" jsonschema:"frase grande dos primeiros segundos, de 4 a 40 letras, sem travessão"`
	Destaque []string           `json:"destaque,omitempty" jsonschema:"palavras do gancho pintadas de verde, até 4"`
	Selos    []string           `json:"selos,omitempty" jsonschema:"até 2 selos curtos, até 22 letras cada (ex.: Em português, Lacrado). Na contagem o primeiro vira o selo de baixo"`
	Produtos []pecaProdutoInput `json:"produtos,omitempty" jsonschema:"produtos do site; o preço e a foto vêm do site, nunca digite preço"`
	ParaODia string             `json:"para_o_dia,omitempty" jsonschema:"só na contagem: o dia em que a peça vai ao ar, AAAA-MM-DD (padrão: hoje). Os dias que faltam são contados daí"`
	Final    pecaFinalInput     `json:"final"`
}

type pecaPreco struct {
	Produto string  `json:"produto"`
	Preco   float64 `json:"preco_pix"`
	Cartao  float64 `json:"preco_cartao,omitempty"`
	Parcela int     `json:"parcelas,omitempty"`
	Link    string  `json:"link"`
}

type pecaOutput struct {
	Arquivo  string      `json:"arquivo"`
	Previas  []string    `json:"previas,omitempty"`
	Segundos float64     `json:"segundos,omitempty"`
	Precos   []pecaPreco `json:"precos,omitempty"`
	Dias     *int        `json:"dias_para_inauguracao,omitempty"`
	Avisos   []string    `json:"avisos,omitempty"`
	Proximo  string      `json:"proximo"`
}

type pecaProduto struct {
	Nome      string  `json:"nome"`
	Imagem    string  `json:"imagem"`
	Proporcao float64 `json:"proporcao"`
	Preco     float64 `json:"preco"`
	Cartao    float64 `json:"cartao,omitempty"`
	Parcelas  int     `json:"parcelas,omitempty"`
}

type pecaFinal struct {
	Titulo   string   `json:"titulo"`
	Destaque []string `json:"destaque"`
	Oferta   string   `json:"oferta"`
	Linhas   []string `json:"linhas"`
	CTA      string   `json:"cta"`
}

type pecaProps struct {
	Music    bool          `json:"music"`
	Modelo   string        `json:"modelo"`
	Etiqueta string        `json:"etiqueta"`
	Gancho   string        `json:"gancho"`
	Destaque []string      `json:"destaque"`
	Selos    []string      `json:"selos"`
	Produtos []pecaProduto `json:"produtos"`
	Dias     int           `json:"dias"`
	Final    pecaFinal     `json:"final"`
}

type renderResult struct {
	Saida    string   `json:"saida"`
	Previas  []string `json:"previas"`
	Segundos float64  `json:"segundos"`
}

type renderFunc func(ctx context.Context, propsPath, kind, output string) (renderResult, error)

type feedItem struct {
	Title        string
	Description  string
	Availability string
	Price        float64
	Card         float64
	Link         string
	Image        string
}

type feedCache struct {
	at    time.Time
	items map[string]feedItem
}

func (a *app) addPecaTool(s *mcp.Server) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "criar_peca",
		Description: "Só para o Social Collecta. Gera o vídeo (Reels e TikTok) ou a arte (post) da Collecta a partir de um modelo pronto: produto, carta ou contagem para a inauguração. Preço e foto vêm do site; você escreve só os textos. Devolve o caminho do arquivo e as prévias em PNG: olhe as prévias com Read antes de pedir aprovação. Uma peça por vez, até 30 por dia somando criar_peca e criar_lista.",
	}, a.toolPeca)
}

func (a *app) toolPeca(ctx context.Context, _ *mcp.CallToolRequest, in pecaInput) (*mcp.CallToolResult, pecaOutput, error) {
	modelo := strings.ToLower(strings.TrimSpace(in.Modelo))
	formato := strings.ToLower(strings.TrimSpace(in.Formato))
	ext, ok := pecaKinds[formato]
	if !pecaModels[modelo] || !ok {
		return nil, pecaOutput{}, fmt.Errorf("modelo %q ou formato %q não existe: modelo produto, carta ou contagem; formato video ou arte", in.Modelo, in.Formato)
	}
	props, err := a.pecaTexts(modelo, in)
	if err != nil {
		return nil, pecaOutput{}, err
	}
	if err := checkProductCount(modelo, len(in.Produtos)); err != nil {
		return nil, pecaOutput{}, err
	}
	out := pecaOutput{}
	if modelo == "contagem" {
		dias, err := a.daysLeft(in.ParaODia)
		if err != nil {
			return nil, pecaOutput{}, err
		}
		props.Dias = dias
		out.Dias = &dias
	}
	if !a.renderMu.TryLock() {
		return nil, pecaOutput{}, errors.New("outra peça está sendo gerada agora: tente de novo em 1 minuto")
	}
	defer a.renderMu.Unlock()
	if n := a.rendersToday(); n >= rendersPerDay {
		return nil, pecaOutput{}, fmt.Errorf("já foram %d peças hoje, o limite do dia: use as que já existem", n)
	}
	id := randomID()
	a.pruneStaging()
	for i, p := range in.Produtos {
		prod, price, err := a.resolveProduct(ctx, id, i, p)
		if err != nil {
			return nil, pecaOutput{}, err
		}
		props.Produtos = append(props.Produtos, prod)
		out.Precos = append(out.Precos, price)
		if modelo == "carta" && strings.HasPrefix(prod.Imagem, "agente/") {
			out.Avisos = append(out.Avisos, fmt.Sprintf("a foto de %s veio do site: foto de carta avulsa costuma ter a marca SAMPLE. Se a prévia mostrar SAMPLE, não peça aprovação; anote em Pendências do Jairo a foto que falta", prod.Nome))
		}
	}
	name := fmt.Sprintf("%s-%s-%s-%s%s", a.now().In(saoPaulo).Format("2006-01-02"), modelo, slugify(props.Gancho), id, ext)
	output := filepath.Join(a.cfg.MotionDir, "out", "agente", name)
	propsPath := filepath.Join(a.cfg.DataDir, "pecas", id+".json")
	if err := writeJSON(propsPath, props); err != nil {
		return nil, pecaOutput{}, err
	}
	rctx, cancel := context.WithTimeout(ctx, renderTimeout)
	defer cancel()
	res, err := a.render(rctx, propsPath, formato, output)
	if err != nil {
		a.pecaLog(fmt.Sprintf("falhou %s %s %s: %v", id, modelo, formato, err))
		return nil, pecaOutput{}, fmt.Errorf("a peça não renderizou: %v", err)
	}
	a.pecaLog(fmt.Sprintf("gerada %s %s %s %s", id, modelo, formato, output))
	out.Arquivo = output
	out.Previas = res.Previas
	out.Segundos = res.Segundos
	out.Proximo = "olhe as prévias (ou a arte) com Read. Se estiver boa, peça aprovação com pedir_aprovacao usando este arquivo e uma legenda com os mesmos preços; se não, gere de novo com textos melhores."
	if formato == "arte" {
		out.Previas = []string{output}
	}
	return nil, out, nil
}

func checkProductCount(modelo string, n int) error {
	switch {
	case modelo == "carta" && n != 1:
		return errors.New("carta leva exatamente 1 produto (a carta do site)")
	case modelo == "produto" && (n < 1 || n > productsMax):
		return fmt.Errorf("produto leva de 1 a %d produtos", productsMax)
	case modelo == "contagem" && n > productsMax:
		return fmt.Errorf("contagem leva até %d produtos", productsMax)
	}
	return nil
}

func (a *app) pecaTexts(modelo string, in pecaInput) (pecaProps, error) {
	props := pecaProps{
		Music:    true,
		Modelo:   modelo,
		Etiqueta: cleanLine(in.Etiqueta),
		Gancho:   cleanLine(in.Gancho),
		Destaque: cleanList(in.Destaque),
		Selos:    cleanList(in.Selos),
		Produtos: []pecaProduto{},
		Final: pecaFinal{
			Titulo:   cleanLine(in.Final.Titulo),
			Destaque: cleanList(in.Final.Destaque),
			Oferta:   cleanLine(in.Final.Oferta),
			Linhas:   cleanList(in.Final.Linhas),
			CTA:      cleanLine(in.Final.CTA),
		},
	}
	limits := []textLimit{
		{"etiqueta", props.Etiqueta, 1, 32},
		{"gancho", props.Gancho, 4, 40},
		{"final.titulo", props.Final.Titulo, 1, 24},
		{"final.oferta", props.Final.Oferta, 1, 28},
		{"final.cta", props.Final.CTA, 1, 18},
	}
	for _, s := range props.Selos {
		limits = append(limits, textLimit{"selos", s, 1, 22})
	}
	for _, s := range props.Final.Linhas {
		limits = append(limits, textLimit{"final.linhas", s, 1, 32})
	}
	for _, p := range in.Produtos {
		if n := cleanLine(p.Nome); n != "" {
			limits = append(limits, textLimit{"produtos.nome", n, 1, 28})
		}
	}
	for _, l := range limits {
		if err := l.check(); err != nil {
			return pecaProps{}, err
		}
	}
	switch {
	case len(props.Destaque) > 4 || len(props.Final.Destaque) > 4:
		return pecaProps{}, errors.New("destaque leva até 4 palavras")
	case len(props.Selos) > 2:
		return pecaProps{}, errors.New("até 2 selos")
	case len(props.Final.Linhas) < 1 || len(props.Final.Linhas) > 3:
		return pecaProps{}, errors.New("final.linhas leva de 1 a 3 linhas")
	}
	return props, nil
}

func (l textLimit) check() error {
	n := utf8.RuneCountInString(l.value)
	switch {
	case n < l.min:
		return fmt.Errorf("%s vazio", l.field)
	case n > l.max:
		return fmt.Errorf("%s com %d letras: o máximo é %d (%q)", l.field, n, l.max, l.value)
	case strings.ContainsAny(l.value, "—–"):
		return fmt.Errorf("%s com travessão: troque por vírgula ou dois-pontos", l.field)
	}
	return nil
}

func cleanList(in []string) []string {
	out := []string{}
	for _, s := range in {
		if s = cleanLine(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (a *app) daysLeft(raw string) (int, error) {
	today := a.now().In(saoPaulo)
	day := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, saoPaulo)
	if raw = strings.TrimSpace(raw); raw != "" {
		parsed, err := time.ParseInLocation("2006-01-02", raw, saoPaulo)
		if err != nil {
			return 0, fmt.Errorf("para_o_dia inválido %q: use AAAA-MM-DD", raw)
		}
		if parsed.Before(day) {
			return 0, errors.New("para_o_dia já passou")
		}
		day = parsed
	}
	dias := int(inauguration.Sub(day).Hours() / 24)
	switch {
	case dias < 0:
		return 0, errors.New("a inauguração já foi em 15/11: o modelo contagem não vale mais, use produto ou carta")
	case dias > countdownLimit:
		return 0, fmt.Errorf("faltam %d dias: a contagem só vale nos últimos %d", dias, countdownLimit)
	}
	return dias, nil
}

func (a *app) resolveProduct(ctx context.Context, id string, index int, in pecaProdutoInput) (pecaProduto, pecaPreco, error) {
	link, err := normalizeProductLink(in.Link)
	if err != nil {
		return pecaProduto{}, pecaPreco{}, err
	}
	items, err := a.loadFeed(ctx)
	if err != nil {
		return pecaProduto{}, pecaPreco{}, err
	}
	item, ok := items[link]
	if !ok {
		return pecaProduto{}, pecaPreco{}, fmt.Errorf("produto %s não está no catálogo do site: confira o link", link)
	}
	if item.Availability != "in stock" && item.Availability != "preorder" {
		return pecaProduto{}, pecaPreco{}, fmt.Errorf("produto %s está sem estoque no site: escolha outro", link)
	}
	name := cleanLine(in.Nome)
	if name == "" {
		name = shortTitle(item.Title)
		if utf8.RuneCountInString(name) > 28 {
			return pecaProduto{}, pecaPreco{}, fmt.Errorf("o nome do site é longo (%q): passe um nome curto de até 28 letras", name)
		}
	}
	installments := 0
	if m := installmentRe.FindStringSubmatch(item.Description); m != nil {
		installments, _ = strconv.Atoi(m[1])
	}
	prod := pecaProduto{Nome: name, Preco: item.Price, Cartao: item.Card, Parcelas: installments}
	if img := strings.TrimSpace(in.Imagem); img != "" {
		prod.Imagem, prod.Proporcao, err = a.localPhoto(img)
	} else {
		prod.Imagem, prod.Proporcao, err = a.downloadPhoto(ctx, id, index, item.Image)
	}
	if err != nil {
		return pecaProduto{}, pecaPreco{}, err
	}
	return prod, pecaPreco{Produto: name, Preco: item.Price, Cartao: item.Card, Parcela: installments, Link: link}, nil
}

func normalizeProductLink(raw string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || (u.Host != siteHost && u.Host != strings.TrimPrefix(siteHost, "www.")) || !productPath.MatchString(u.Path) {
		return "", fmt.Errorf("link %q não é de um produto do site (https://%s/selado/..., /carta/... ou /acessorio/...)", raw, siteHost)
	}
	return "https://" + siteHost + u.Path, nil
}

func shortTitle(title string) string {
	if i := strings.Index(title, " — "); i > 0 {
		title = title[:i]
	}
	return cleanLine(title)
}

func (a *app) loadFeed(ctx context.Context) (map[string]feedItem, error) {
	a.feedMu.Lock()
	defer a.feedMu.Unlock()
	if a.feed.items != nil && a.now().Sub(a.feed.at) < feedTTL {
		return a.feed.items, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.cfg.FeedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Collecta/1.0")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("catálogo do site fora do ar: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catálogo do site respondeu %d", resp.StatusCode)
	}
	items, err := parseFeed(io.LimitReader(resp.Body, feedMax))
	if err != nil {
		return nil, err
	}
	if err := a.addCardPrices(ctx, items); err != nil {
		return nil, err
	}
	a.feed = feedCache{at: a.now(), items: items}
	return items, nil
}

func (a *app) addCardPrices(ctx context.Context, items map[string]feedItem) error {
	var catalog struct {
		Items []struct {
			Slug    string  `json:"slug"`
			CardBRL float64 `json:"cardBRL"`
		} `json:"items"`
	}
	if err := a.call(ctx, http.MethodGet, "/api/storefront", nil, &catalog); err != nil {
		return fmt.Errorf("preço no cartão indisponível: %v", err)
	}
	cards := map[string]float64{}
	for _, it := range catalog.Items {
		if it.Slug != "" && it.CardBRL > 0 {
			cards[it.Slug] = it.CardBRL
		}
	}
	for link, item := range items {
		item.Card = cards[path.Base(link)]
		items[link] = item
	}
	return nil
}

func parseFeed(r io.Reader) (map[string]feedItem, error) {
	rows, err := csv.NewReader(r).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("catálogo do site ilegível: %v", err)
	}
	if len(rows) < 2 {
		return nil, errors.New("catálogo do site vazio")
	}
	col := map[string]int{}
	for i, name := range rows[0] {
		col[name] = i
	}
	for _, need := range []string{"title", "description", "availability", "price", "link", "image_link"} {
		if _, ok := col[need]; !ok {
			return nil, fmt.Errorf("catálogo do site sem a coluna %s", need)
		}
	}
	items := map[string]feedItem{}
	for _, row := range rows[1:] {
		if len(row) != len(rows[0]) {
			continue
		}
		price, err := strconv.ParseFloat(strings.TrimSuffix(row[col["price"]], " BRL"), 64)
		if err != nil {
			continue
		}
		link, err := normalizeProductLink(row[col["link"]])
		if err != nil {
			continue
		}
		items[link] = feedItem{
			Title: row[col["title"]], Description: row[col["description"]], Availability: row[col["availability"]],
			Price: price, Link: link, Image: row[col["image_link"]],
		}
	}
	return items, nil
}

func (a *app) localPhoto(rel string) (string, float64, error) {
	if !localImage.MatchString(rel) {
		return "", 0, fmt.Errorf("imagem %q inválida: use products/<arquivo> ou cards/<arquivo> de store/motion/public", rel)
	}
	ratio, err := imageRatio(filepath.Join(a.cfg.MotionDir, "public", rel))
	if err != nil {
		return "", 0, fmt.Errorf("imagem %s: %v", rel, err)
	}
	return rel, ratio, nil
}

func imageRatio(path string) (float64, error) {
	fh, err := os.Open(path)
	if err != nil {
		return 0, errors.New("não existe")
	}
	defer fh.Close()
	if strings.EqualFold(filepath.Ext(path), ".webp") {
		w, h, err := webpSize(fh)
		if err != nil {
			return 0, err
		}
		return float64(w) / float64(h), nil
	}
	cfg, _, err := image.DecodeConfig(fh)
	if err != nil || cfg.Height == 0 {
		return 0, errors.New("não é jpg nem png válido")
	}
	return float64(cfg.Width) / float64(cfg.Height), nil
}

func webpSize(r io.Reader) (int, int, error) {
	head := make([]byte, 30)
	if _, err := io.ReadFull(r, head); err != nil || string(head[0:4]) != "RIFF" || string(head[8:12]) != "WEBP" {
		return 0, 0, errors.New("webp inválido")
	}
	switch string(head[12:16]) {
	case "VP8X":
		w := 1 + (int(head[24]) | int(head[25])<<8 | int(head[26])<<16)
		h := 1 + (int(head[27]) | int(head[28])<<8 | int(head[29])<<16)
		return w, h, nil
	case "VP8 ":
		w := (int(head[26]) | int(head[27])<<8) & 0x3fff
		h := (int(head[28]) | int(head[29])<<8) & 0x3fff
		return w, h, nil
	case "VP8L":
		b := uint32(head[21]) | uint32(head[22])<<8 | uint32(head[23])<<16 | uint32(head[24])<<24
		return int(b&0x3fff) + 1, int((b>>14)&0x3fff) + 1, nil
	}
	return 0, 0, errors.New("webp em formato desconhecido")
}

func (a *app) downloadPhoto(ctx context.Context, id string, index int, raw string) (string, float64, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return "", 0, errors.New("o produto não tem foto https no site: passe uma imagem local")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", "Collecta/1.0")
	resp, err := a.client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("foto do produto não baixou: %v", err)
	}
	defer resp.Body.Close()
	ext, ok := photoTypes[strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])]
	if resp.StatusCode != http.StatusOK || !ok {
		return "", 0, fmt.Errorf("foto do produto não baixou (%d %s): passe uma imagem local", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, photoMax+1))
	if err != nil {
		return "", 0, err
	}
	if len(data) > photoMax {
		return "", 0, errors.New("foto do produto passa de 15 MB")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Height == 0 {
		return "", 0, errors.New("foto do produto ilegível: passe uma imagem local")
	}
	rel := filepath.Join("agente", id, fmt.Sprintf("%d%s", index+1, ext))
	dst := filepath.Join(a.cfg.MotionDir, "public", rel)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", 0, err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return "", 0, err
	}
	return filepath.ToSlash(rel), float64(cfg.Width) / float64(cfg.Height), nil
}

func (a *app) pruneStaging() {
	root := filepath.Join(a.cfg.MotionDir, "public", "agente")
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		info, err := e.Info()
		if err == nil && a.now().Sub(info.ModTime()) > stagingKeep {
			_ = os.RemoveAll(filepath.Join(root, e.Name()))
		}
	}
}

func (a *app) rendersToday() int {
	fh, err := os.Open(filepath.Join(a.cfg.DataDir, "pecas.log"))
	if err != nil {
		return 0
	}
	defer fh.Close()
	today := a.now().In(saoPaulo).Format("2006-01-02")
	n := 0
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		if strings.HasPrefix(sc.Text(), today+" ") && strings.Contains(sc.Text(), " gerada ") {
			n++
		}
	}
	return n
}

func (a *app) pecaLog(line string) {
	f, err := os.OpenFile(filepath.Join(a.cfg.DataDir, "pecas.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s %s\n", a.now().In(saoPaulo).Format("2006-01-02 15:04"), line)
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func randomID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c").Replace(s)
	s = strings.Trim(slugWord.ReplaceAllString(s, "-"), "-")
	if len(s) > 32 {
		s = strings.TrimRight(s[:32], "-")
	}
	if s == "" {
		return "peca"
	}
	return s
}

func nodeRender(node, motionDir string) renderFunc {
	return func(ctx context.Context, propsPath, kind, output string) (renderResult, error) {
		cmd := exec.CommandContext(ctx, node, "scripts/peca.mjs", propsPath, kind, output)
		cmd.Dir = motionDir
		cmd.Env = append(os.Environ(), "DYLD_LIBRARY_PATH="+filepath.Join(motionDir, "node_modules", "@remotion", "compositor-darwin-arm64"))
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			msg := strings.TrimSpace(stderr.String())
			if len(msg) > 400 {
				msg = msg[len(msg)-400:]
			}
			return renderResult{}, fmt.Errorf("%v: %s", err, msg)
		}
		lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		var res renderResult
		if err := json.Unmarshal([]byte(lines[len(lines)-1]), &res); err != nil {
			return renderResult{}, fmt.Errorf("saída do render ilegível: %v", err)
		}
		return res, nil
	}
}
