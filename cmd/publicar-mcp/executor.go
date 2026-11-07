package main

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"opdeals/internal/adtestq"
)

const (
	pauseMinSpend   = 15.0
	pauseRatio      = 2.0
	minActiveAds    = 2
	finishGrace     = time.Hour
	launchDelay     = 5 * time.Minute
	videoPollEvery  = 10 * time.Second
	videoPollMax    = 10 * time.Minute
	storeLatitude   = -16.6799
	storeLongitude  = -49.2659
	storeRadiusKm   = 30
	audienceMinAge  = 18
	audienceMaxAge  = 45
	metaErrorMaxLen = 300
)

var brazilInterests = []string{"Pokémon", "One Piece", "Trading card game"}

type metaConfig struct {
	Token      string
	Account    string
	PageID     string
	IGUserID   string
	IGUsername string
	PixelID    string
	BaseURL    string
	RuploadURL string
}

func loadMetaConfig(getenv func(string) string) (metaConfig, error) {
	version := cmp.Or(getenv("META_VERSAO"), "v27.0")
	cfg := metaConfig{
		Token:      getenv("META_TOKEN"),
		Account:    getenv("META_CONTA"),
		PageID:     getenv("META_PAGINA"),
		IGUserID:   getenv("META_INSTAGRAM_ID"),
		IGUsername: cmp.Or(getenv("META_INSTAGRAM_USUARIO"), "tcgcollecta"),
		PixelID:    getenv("META_PIXEL"),
		BaseURL:    strings.TrimRight(cmp.Or(getenv("META_URL"), "https://graph.facebook.com"), "/") + "/" + version,
		RuploadURL: strings.TrimRight(cmp.Or(getenv("META_RUPLOAD"), "https://rupload.facebook.com/ig-api-upload"), "/") + "/" + version,
	}
	for name, v := range map[string]string{"META_TOKEN": cfg.Token, "META_CONTA": cfg.Account, "META_PAGINA": cfg.PageID, "META_INSTAGRAM_ID": cfg.IGUserID} {
		if v == "" {
			return cfg, fmt.Errorf("%s não definido em anuncios.env", name)
		}
	}
	if !strings.HasPrefix(cfg.Account, "act_") {
		cfg.Account = "act_" + cfg.Account
	}
	return cfg, nil
}

func readEnvFile(path string) (map[string]string, error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	out := map[string]string{}
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	return out, sc.Err()
}

type graph struct {
	cfg    metaConfig
	client *http.Client
}

type graphError struct {
	Error struct {
		Message      string `json:"message"`
		UserMessage  string `json:"error_user_msg"`
		Code         int    `json:"code"`
		ErrorSubcode int    `json:"error_subcode"`
	} `json:"error"`
}

func (g *graph) do(req *http.Request, out any) error {
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("Meta fora do ar: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		var ge graphError
		_ = json.Unmarshal(body, &ge)
		msg := cmp.Or(ge.Error.UserMessage, ge.Error.Message, strings.TrimSpace(string(body)))
		if len(msg) > metaErrorMaxLen {
			msg = msg[:metaErrorMaxLen]
		}
		return fmt.Errorf("Meta recusou (%d/%d): %s", ge.Error.Code, ge.Error.ErrorSubcode, msg)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

func (g *graph) post(ctx context.Context, path string, params url.Values, out any) error {
	params.Set("access_token", g.cfg.Token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.cfg.BaseURL+"/"+path, strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return g.do(req, out)
}

func (g *graph) get(ctx context.Context, path string, params url.Values, out any) error {
	params.Set("access_token", g.cfg.Token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.cfg.BaseURL+"/"+path+"?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	return g.do(req, out)
}

func (g *graph) upload(ctx context.Context, path, field, file string, params url.Values, out any) error {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	params.Set("access_token", g.cfg.Token)
	for k, vs := range params {
		for _, v := range vs {
			if err := mw.WriteField(k, v); err != nil {
				return err
			}
		}
	}
	part, err := mw.CreateFormFile(field, filepath.Base(file))
	if err != nil {
		return err
	}
	fh, err := os.Open(file)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, fh)
	fh.Close()
	if err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.cfg.BaseURL+"/"+path, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return g.do(req, out)
}

func jsonParam(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}

type executor struct {
	app       *app
	g         *graph
	pollEvery time.Duration
	pollMax   time.Duration
}

func newExecutor(a *app, mc metaConfig, client *http.Client) *executor {
	return &executor{app: a, g: &graph{cfg: mc, client: client}, pollEvery: videoPollEvery, pollMax: videoPollMax}
}

func (e *executor) list(ctx context.Context, status string) ([]adtestq.Test, error) {
	var list struct {
		Items []adtestq.Test `json:"items"`
	}
	if err := e.app.call(ctx, http.MethodGet, "/api/adtests?status="+status, nil, &list); err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (e *executor) run(ctx context.Context) error {
	errs := e.publishTrials(ctx)
	approved, err := e.list(ctx, adtestq.StatusApproved)
	if err != nil {
		return errors.Join(append(errs, err)...)
	}
	for _, t := range approved {
		if err := e.launch(ctx, t); err != nil {
			errs = append(errs, fmt.Errorf("teste %s: %w", t.ID, err))
		}
	}
	running, err := e.list(ctx, adtestq.StatusRunning)
	if err != nil {
		return errors.Join(append(errs, err)...)
	}
	for _, t := range running {
		if err := e.review(ctx, t); err != nil {
			errs = append(errs, fmt.Errorf("teste %s: %w", t.ID, err))
		}
	}
	return errors.Join(errs...)
}

func (e *executor) notify(ctx context.Context, title, priority, tags, text string) {
	if err := e.app.push(ctx, map[string]string{"Title": title, "Priority": priority, "Tags": tags}, strings.NewReader(text)); err != nil {
		e.app.adLog("aviso não chegou: " + err.Error())
	}
}

func (e *executor) launch(ctx context.Context, t adtestq.Test) error {
	meta := adtestq.Meta{}
	endsAt, err := e.create(ctx, t, &meta)
	if err != nil {
		if meta.CampaignID != "" {
			if perr := e.g.post(ctx, meta.CampaignID, url.Values{"status": {"PAUSED"}}, nil); perr != nil {
				err = fmt.Errorf("%v (e a campanha %s não pausou: %v)", err, meta.CampaignID, perr)
			}
		}
		e.app.adLog(fmt.Sprintf("falhou ao subir %s: %v", t.ID, err))
		var failedMeta *adtestq.Meta
		if meta.CampaignID != "" {
			failedMeta = &meta
		}
		if cerr := e.app.call(ctx, http.MethodPost, "/api/adtests/"+url.PathEscape(t.ID)+"/failed", map[string]any{"error": err.Error(), "meta": failedMeta}, nil); cerr != nil {
			return errors.Join(err, cerr)
		}
		e.notify(ctx, "Teste de anúncio não subiu", "high", "warning", fmt.Sprintf("%s %s: %v. Nada foi gasto além do que o Meta mostrar na campanha.", objectiveLabel[t.Objective], reais(t.BudgetCents), err))
		return err
	}
	if err := e.app.call(ctx, http.MethodPost, "/api/adtests/"+url.PathEscape(t.ID)+"/started", map[string]any{"meta": meta, "endsAt": endsAt}, nil); err != nil {
		return err
	}
	e.app.adLog(fmt.Sprintf("no ar %s campanha %s conjunto %s %d anúncios até %s", t.ID, meta.CampaignID, meta.AdSetID, len(meta.Ads), endsAt.In(saoPaulo).Format("02/01 15:04")))
	e.notify(ctx, "Teste de anúncio no ar", "default", "rocket", fmt.Sprintf("%s, %s, %d criativos, %s até %s.", objectiveLabel[t.Objective], audienceLabel[t.Audience], len(meta.Ads), reais(t.BudgetCents), endsAt.In(saoPaulo).Format("02/01 15:04")))
	return nil
}

func (e *executor) create(ctx context.Context, t adtestq.Test, meta *adtestq.Meta) (time.Time, error) {
	for _, c := range t.Creatives {
		got, err := hashFile(c.Path)
		if err != nil {
			return time.Time{}, err
		}
		if got.SHA256 != c.SHA256 {
			return time.Time{}, fmt.Errorf("%s mudou depois da aprovação", filepath.Base(c.Path))
		}
	}
	if t.Objective == "vendas" && e.g.cfg.PixelID == "" {
		return time.Time{}, errors.New("META_PIXEL não definido em anuncios.env: teste de vendas precisa do Pixel")
	}
	targeting, err := e.targeting(ctx, t.Audience)
	if err != nil {
		return time.Time{}, err
	}
	act := e.g.cfg.Account
	objective, goal, destination := "OUTCOME_TRAFFIC", "PROFILE_VISIT", "INSTAGRAM_PROFILE"
	promoted := map[string]string{"page_id": e.g.cfg.PageID}
	if t.Objective == "vendas" {
		objective, goal, destination = "OUTCOME_SALES", "OFFSITE_CONVERSIONS", "WEBSITE"
		promoted = map[string]string{"pixel_id": e.g.cfg.PixelID, "custom_event_type": "PURCHASE"}
	}
	var created struct {
		ID string `json:"id"`
	}
	name := fmt.Sprintf("Collecta teste %s %s", t.ID, t.Objective)
	if err := e.g.post(ctx, act+"/campaigns", url.Values{
		"name": {name}, "objective": {objective}, "status": {"PAUSED"},
		"special_ad_categories": {"[]"}, "is_adset_budget_sharing_enabled": {"false"},
	}, &created); err != nil {
		return time.Time{}, err
	}
	meta.CampaignID = created.ID
	start := e.app.now().Add(launchDelay).Truncate(time.Minute)
	endsAt := start.Add(time.Duration(t.Days) * 24 * time.Hour)
	if err := e.g.post(ctx, act+"/adsets", url.Values{
		"name": {name}, "campaign_id": {meta.CampaignID}, "status": {"PAUSED"},
		"lifetime_budget": {strconv.FormatInt(t.BudgetCents, 10)},
		"start_time":      {start.In(saoPaulo).Format(time.RFC3339)}, "end_time": {endsAt.In(saoPaulo).Format(time.RFC3339)},
		"billing_event": {"IMPRESSIONS"}, "bid_strategy": {"LOWEST_COST_WITHOUT_CAP"},
		"optimization_goal": {goal}, "destination_type": {destination},
		"promoted_object": {jsonParam(promoted)}, "targeting": {jsonParam(targeting)},
	}, &created); err != nil {
		return time.Time{}, err
	}
	meta.AdSetID = created.ID
	for i, c := range t.Creatives {
		creativeID, err := e.creative(ctx, t, i, c)
		if err != nil {
			return time.Time{}, fmt.Errorf("criativo %d: %w", i+1, err)
		}
		if err := e.g.post(ctx, act+"/ads", url.Values{
			"name": {fmt.Sprintf("%s %d %s", t.ID, i+1, c.Hook)}, "adset_id": {meta.AdSetID},
			"creative": {jsonParam(map[string]string{"creative_id": creativeID})}, "status": {"PAUSED"},
		}, &created); err != nil {
			return time.Time{}, fmt.Errorf("criativo %d: %w", i+1, err)
		}
		meta.Ads = append(meta.Ads, adtestq.MetaAd{ID: created.ID, Creative: i})
	}
	ids := make([]string, 0, len(meta.Ads)+2)
	for _, ad := range meta.Ads {
		ids = append(ids, ad.ID)
	}
	for _, id := range append(ids, meta.AdSetID, meta.CampaignID) {
		if err := e.g.post(ctx, id, url.Values{"status": {"ACTIVE"}}, nil); err != nil {
			return time.Time{}, err
		}
	}
	return endsAt, nil
}

func (e *executor) targeting(ctx context.Context, audience string) (map[string]any, error) {
	base := map[string]any{"age_min": audienceMinAge, "age_max": audienceMaxAge}
	if audience == "goiania" {
		base["geo_locations"] = map[string]any{"custom_locations": []map[string]any{{
			"latitude": storeLatitude, "longitude": storeLongitude, "radius": storeRadiusKm, "distance_unit": "kilometer",
		}}}
		base["targeting_automation"] = map[string]int{"advantage_audience": 0}
		return base, nil
	}
	base["geo_locations"] = map[string]any{"countries": []string{"BR"}}
	var interests []map[string]string
	for _, q := range brazilInterests {
		var found struct {
			Data []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"data"`
		}
		if err := e.g.get(ctx, "search", url.Values{"type": {"adinterest"}, "q": {q}, "limit": {"1"}}, &found); err != nil {
			return nil, err
		}
		if len(found.Data) > 0 {
			interests = append(interests, map[string]string{"id": found.Data[0].ID, "name": found.Data[0].Name})
		}
	}
	if len(interests) == 0 {
		base["targeting_automation"] = map[string]int{"advantage_audience": 1}
		return base, nil
	}
	base["flexible_spec"] = []map[string]any{{"interests": interests}}
	base["targeting_automation"] = map[string]int{"advantage_audience": 0}
	return base, nil
}

func (e *executor) imageHash(ctx context.Context, file string) (string, error) {
	var res struct {
		Images map[string]struct {
			Hash string `json:"hash"`
		} `json:"images"`
	}
	if err := e.g.upload(ctx, e.g.cfg.Account+"/adimages", "filename", file, url.Values{}, &res); err != nil {
		return "", err
	}
	for _, img := range res.Images {
		if img.Hash != "" {
			return img.Hash, nil
		}
	}
	return "", errors.New("Meta não devolveu o hash da imagem")
}

func (e *executor) video(ctx context.Context, t adtestq.Test, index int, file string) (string, error) {
	var created struct {
		ID string `json:"id"`
	}
	if err := e.g.upload(ctx, e.g.cfg.Account+"/advideos", "source", file, url.Values{"name": {fmt.Sprintf("%s %d", t.ID, index+1)}}, &created); err != nil {
		return "", err
	}
	deadline := time.Now().Add(e.pollMax)
	for {
		var st struct {
			Status struct {
				VideoStatus string `json:"video_status"`
			} `json:"status"`
		}
		if err := e.g.get(ctx, created.ID, url.Values{"fields": {"status"}}, &st); err != nil {
			return "", err
		}
		switch st.Status.VideoStatus {
		case "ready":
			return created.ID, nil
		case "error":
			return "", errors.New("o Meta não conseguiu processar o vídeo")
		}
		if time.Now().After(deadline) {
			return "", errors.New("o vídeo não ficou pronto no Meta a tempo: tento de novo só se o Social propuser outra vez")
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(e.pollEvery):
		}
	}
}

func (e *executor) creative(ctx context.Context, t adtestq.Test, index int, c adtestq.Creative) (string, error) {
	cta := map[string]any{"type": "VIEW_INSTAGRAM_PROFILE", "value": map[string]string{"link": "https://www.instagram.com/" + e.g.cfg.IGUsername}}
	link := "https://www.instagram.com/" + e.g.cfg.IGUsername
	if t.Objective == "vendas" {
		link = t.Link
		cta = map[string]any{"type": "SHOP_NOW", "value": map[string]string{"link": link}}
	}
	spec := map[string]any{"page_id": e.g.cfg.PageID, "instagram_user_id": e.g.cfg.IGUserID}
	if c.Kind == "video" {
		hash, err := e.imageHash(ctx, c.Thumb)
		if err != nil {
			return "", err
		}
		videoID, err := e.video(ctx, t, index, c.Path)
		if err != nil {
			return "", err
		}
		data := map[string]any{"video_id": videoID, "image_hash": hash, "message": c.Text, "call_to_action": cta}
		if c.Headline != "" {
			data["title"] = c.Headline
		}
		spec["video_data"] = data
	} else {
		hash, err := e.imageHash(ctx, c.Path)
		if err != nil {
			return "", err
		}
		data := map[string]any{"image_hash": hash, "link": link, "message": c.Text, "call_to_action": cta}
		if c.Headline != "" {
			data["name"] = c.Headline
		}
		spec["link_data"] = data
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := e.g.post(ctx, e.g.cfg.Account+"/adcreatives", url.Values{
		"name": {fmt.Sprintf("%s %d %s", t.ID, index+1, c.Hook)}, "object_story_spec": {jsonParam(spec)},
	}, &created); err != nil {
		return "", err
	}
	return created.ID, nil
}

type adResult struct {
	index   int
	spend   float64
	results float64
}

func (r adResult) cost() float64 {
	if r.results == 0 {
		return 0
	}
	return r.spend / r.results
}

type insightRow struct {
	AdID             string `json:"ad_id"`
	Spend            string `json:"spend"`
	Clicks           string `json:"clicks"`
	InlineLinkClicks string `json:"inline_link_clicks"`
	Actions          []struct {
		Type  string `json:"action_type"`
		Value string `json:"value"`
	} `json:"actions"`
}

func number(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func (row insightRow) action(types ...string) float64 {
	for _, want := range types {
		for _, a := range row.Actions {
			if a.Type == want {
				return number(a.Value)
			}
		}
	}
	return 0
}

func measure(objective string, rows []insightRow) (string, func(insightRow) float64) {
	clicks := func(r insightRow) float64 { return max(number(r.InlineLinkClicks), number(r.Clicks)) }
	if objective == "seguidores" {
		return "visitas ao perfil", clicks
	}
	purchases := func(r insightRow) float64 { return r.action("omni_purchase", "offsite_conversion.fb_pixel_purchase") }
	pages := func(r insightRow) float64 { return r.action("landing_page_view") }
	for _, r := range rows {
		if purchases(r) > 0 {
			return "compras", purchases
		}
	}
	for _, r := range rows {
		if pages(r) > 0 {
			return "visitas à página", pages
		}
	}
	return "cliques", clicks
}

func (e *executor) review(ctx context.Context, t adtestq.Test) error {
	if t.Meta == nil {
		return errors.New("teste no ar sem ids do Meta")
	}
	var res struct {
		Data []insightRow `json:"data"`
	}
	if err := e.g.get(ctx, t.Meta.AdSetID+"/insights", url.Values{
		"level": {"ad"}, "date_preset": {"maximum"}, "fields": {"ad_id,spend,clicks,inline_link_clicks,actions"},
	}, &res); err != nil {
		return err
	}
	metric, count := measure(t.Objective, res.Data)
	byAd := map[string]insightRow{}
	for _, r := range res.Data {
		byAd[r.AdID] = r
	}
	results := make([]adResult, len(t.Meta.Ads))
	for i, ad := range t.Meta.Ads {
		r := byAd[ad.ID]
		results[i] = adResult{index: i, spend: number(r.Spend), results: count(r)}
	}
	meta := *t.Meta
	meta.Ads = append([]adtestq.MetaAd(nil), t.Meta.Ads...)
	paused := e.pauseLosers(ctx, &meta, results, metric)
	if len(paused) > 0 {
		if err := e.app.call(ctx, http.MethodPost, "/api/adtests/"+url.PathEscape(t.ID)+"/meta", meta, nil); err != nil {
			return err
		}
		for _, line := range paused {
			e.app.adLog(fmt.Sprintf("pausado %s: %s", t.ID, line))
		}
		e.notify(ctx, "Criativo pausado", "default", "pause_button", strings.Join(paused, "\n"))
	}
	if e.app.now().Before(t.EndsAt.Add(finishGrace)) {
		return nil
	}
	summary := summarize(t, meta, results, metric)
	if err := e.app.call(ctx, http.MethodPost, "/api/adtests/"+url.PathEscape(t.ID)+"/finished", map[string]string{"summary": summary}, nil); err != nil {
		return err
	}
	e.app.adLog(fmt.Sprintf("terminado %s: %s", t.ID, strings.ReplaceAll(summary, "\n", " | ")))
	e.notify(ctx, "Teste de anúncio terminou", "default", "trophy", summary)
	return nil
}

func (e *executor) pauseLosers(ctx context.Context, meta *adtestq.Meta, results []adResult, metric string) []string {
	active := 0
	best := 0.0
	for i, r := range results {
		if meta.Ads[i].Paused {
			continue
		}
		active++
		if r.spend >= pauseMinSpend && r.results > 0 && (best == 0 || r.cost() < best) {
			best = r.cost()
		}
	}
	if best == 0 {
		return nil
	}
	var losers []adResult
	for i, r := range results {
		if meta.Ads[i].Paused || r.spend < pauseMinSpend {
			continue
		}
		if r.results == 0 || r.cost() > pauseRatio*best {
			losers = append(losers, r)
		}
	}
	sort.Slice(losers, func(i, j int) bool {
		ci, cj := losers[i].cost(), losers[j].cost()
		if ci == 0 || cj == 0 {
			return ci == 0 && cj != 0
		}
		return ci > cj
	})
	var lines []string
	for _, r := range losers {
		if active <= minActiveAds {
			break
		}
		ad := &meta.Ads[r.index]
		if err := e.g.post(ctx, ad.ID, url.Values{"status": {"PAUSED"}}, nil); err != nil {
			e.app.adLog(fmt.Sprintf("não pausou %s: %v", ad.ID, err))
			continue
		}
		reason := fmt.Sprintf("R$ %.2f gastos e nenhuma %s", r.spend, metric)
		if r.results > 0 {
			reason = fmt.Sprintf("R$ %.2f por resultado (%s), mais que o dobro do melhor (R$ %.2f)", r.cost(), metric, best)
		}
		ad.Paused = true
		ad.Reason = reason
		active--
		lines = append(lines, fmt.Sprintf("criativo %d: %s", r.index+1, reason))
	}
	return lines
}

func summarize(t adtestq.Test, meta adtestq.Meta, results []adResult, metric string) string {
	var b strings.Builder
	var spent float64
	winner := -1
	for i, r := range results {
		spent += r.spend
		if r.results > 0 && (winner < 0 || r.cost() < results[winner].cost()) {
			winner = i
		}
	}
	fmt.Fprintf(&b, "%s, %s: R$ %.2f gastos de %s, medido em %s.", objectiveLabel[t.Objective], audienceLabel[t.Audience], spent, reais(t.BudgetCents), metric)
	for i, r := range results {
		c := t.Creatives[meta.Ads[i].Creative]
		state := ""
		if meta.Ads[i].Paused {
			state = " (pausado)"
		}
		cost := "sem resultado"
		if r.results > 0 {
			cost = fmt.Sprintf("R$ %.2f cada", r.cost())
		}
		fmt.Fprintf(&b, "\n%d. %s%s: R$ %.2f, %.0f %s, %s", i+1, c.Hook, state, r.spend, r.results, metric, cost)
	}
	if winner >= 0 {
		fmt.Fprintf(&b, "\nVencedor: %d. %s.", winner+1, t.Creatives[meta.Ads[winner].Creative].Hook)
	} else {
		b.WriteString("\nSem vencedor: nenhum criativo trouxe resultado.")
	}
	return b.String()
}
