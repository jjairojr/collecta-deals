package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"opdeals/internal/adtestq"
)

// Criativo "post existente": impulsiona um post, reel ou carrossel que já está no
// @tcgcollecta, com a legenda dele. O anúncio usa source_instagram_media_id, então
// texto e título não existem: vale o que o post já mostra.

const (
	postListPages   = 4 // até 200 posts procurados por link
	postPreviewMax  = 8 << 20
	postCaptionShow = 120
)

var (
	mediaIDPattern   = regexp.MustCompile(`^[0-9]{6,25}$`)
	shortcodePattern = regexp.MustCompile(`^/(?:p|reel|reels|tv)/([A-Za-z0-9_-]+)/?$`)
)

type igMedia struct {
	ID               string         `json:"id"`
	Username         string         `json:"username"`
	MediaType        string         `json:"media_type"`
	MediaProductType string         `json:"media_product_type"`
	Permalink        string         `json:"permalink"`
	Caption          string         `json:"caption"`
	MediaURL         string         `json:"media_url"`
	ThumbnailURL     string         `json:"thumbnail_url"`
	Boost            map[string]any `json:"boost_eligibility_info"`
}

const igMediaFields = "id,username,media_type,media_product_type,permalink,caption,media_url,thumbnail_url,boost_eligibility_info"

// boostRefusal says why Meta will not boost the media, or "" when it can.
func (m igMedia) boostRefusal() string {
	if m.MediaProductType == "STORY" {
		return "é um story, que some em 24 horas; use um post, reel ou carrossel do feed"
	}
	if m.Boost == nil {
		return ""
	}
	if ok, _ := m.Boost["eligible_to_boost"].(bool); ok {
		return ""
	}
	var why []string
	for k, v := range m.Boost {
		if k != "eligible_to_boost" {
			why = append(why, fmt.Sprint(v))
		}
	}
	reason := "o Meta não deixa impulsionar este post"
	if len(why) > 0 {
		reason += " (" + strings.Join(why, "; ") + ")"
	}
	return reason + ". Música com licença, filtros e elementos interativos costumam impedir; escolha outro post"
}

func shortcodeOf(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || strings.TrimPrefix(u.Host, "www.") != "instagram.com" {
		return ""
	}
	m := shortcodePattern.FindStringSubmatch(u.Path)
	if m == nil {
		return ""
	}
	return m[1]
}

// metaGraph reads anuncios.env so the MCP can check a post before asking the owner.
func (a *app) metaGraph() (*graph, error) {
	env, err := readEnvFile(a.cfg.AdsEnv)
	if err != nil {
		return nil, errors.New("post existente precisa do anuncios.env com o token do Meta; use um arquivo")
	}
	cfg, err := loadMetaConfig(func(k string) string { return env[k] })
	if err != nil {
		return nil, fmt.Errorf("post existente precisa do anuncios.env completo: %v", err)
	}
	return &graph{cfg: cfg, client: a.client}, nil
}

// findMedia turns a media id or an instagram.com link into the media of the
// store's account, read from Meta.
func findMedia(ctx context.Context, g *graph, raw string) (igMedia, error) {
	raw = strings.TrimSpace(raw)
	id := raw
	if !mediaIDPattern.MatchString(raw) {
		code := shortcodeOf(raw)
		if code == "" {
			return igMedia{}, fmt.Errorf("post %q: use o id da mídia ou o link do post (https://www.instagram.com/p/... ou /reel/...)", raw)
		}
		var err error
		if id, err = mediaByShortcode(ctx, g, code); err != nil {
			return igMedia{}, err
		}
	}
	var m igMedia
	if err := g.get(ctx, id, url.Values{"fields": {igMediaFields}}, &m); err != nil {
		return igMedia{}, fmt.Errorf("post %s: %v", raw, err)
	}
	if m.Username != "" && !strings.EqualFold(m.Username, g.cfg.IGUsername) {
		return igMedia{}, fmt.Errorf("post %s é de @%s, não do @%s", raw, m.Username, g.cfg.IGUsername)
	}
	return m, nil
}

func mediaByShortcode(ctx context.Context, g *graph, code string) (string, error) {
	params := url.Values{"fields": {"id,permalink"}, "limit": {"50"}}
	for page := 0; page < postListPages; page++ {
		var list struct {
			Data []struct {
				ID        string `json:"id"`
				Permalink string `json:"permalink"`
			} `json:"data"`
			Paging struct {
				Cursors struct {
					After string `json:"after"`
				} `json:"cursors"`
				Next string `json:"next"`
			} `json:"paging"`
		}
		if err := g.get(ctx, g.cfg.IGUserID+"/media", params, &list); err != nil {
			return "", fmt.Errorf("não consegui ler os posts do @%s: %v", g.cfg.IGUsername, err)
		}
		for _, d := range list.Data {
			if shortcodeOf(d.Permalink) == code {
				return d.ID, nil
			}
		}
		if list.Paging.Next == "" || list.Paging.Cursors.After == "" {
			break
		}
		params.Set("after", list.Paging.Cursors.After)
	}
	return "", fmt.Errorf("não achei o post %s entre os últimos posts do @%s", code, g.cfg.IGUsername)
}

// stagePost checks an existing post and keeps its preview for the owner.
func (a *app) stagePost(ctx context.Context, g *graph, stage string, i int, c adCreativeInput, hook string) (adtestq.Creative, error) {
	switch {
	case strings.TrimSpace(c.Arquivo) != "":
		return adtestq.Creative{}, errors.New("use post ou arquivo, não os dois")
	case strings.TrimSpace(c.Texto) != "" || strings.TrimSpace(c.Titulo) != "":
		return adtestq.Creative{}, errors.New("no post existente vale a legenda do post: deixe texto e titulo vazios")
	}
	m, err := findMedia(ctx, g, c.Post)
	if err != nil {
		return adtestq.Creative{}, err
	}
	if why := m.boostRefusal(); why != "" {
		return adtestq.Creative{}, fmt.Errorf("post %s: %s", cmp.Or(m.Permalink, c.Post), why)
	}
	caption := []rune(cleanLine(m.Caption))
	if len(caption) > postCaptionShow {
		caption = append(caption[:postCaptionShow], '…')
	}
	cr := adtestq.Creative{Kind: "post", MediaID: m.ID, Permalink: m.Permalink, Hook: hook,
		Text: strings.TrimSpace(fmt.Sprintf("post %s (%s) %s", m.Permalink, strings.ToLower(m.MediaType), string(caption)))}
	src := m.MediaURL
	if m.ThumbnailURL != "" {
		src = m.ThumbnailURL
	}
	if src != "" {
		dst := filepath.Join(stage, fmt.Sprintf("%02d-capa.jpg", i+1))
		if err := download(ctx, a.client, src, dst); err == nil {
			cr.Path = dst
		}
	}
	return cr, nil
}

func download(ctx context.Context, client *http.Client, src, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, src, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("prévia respondeu %d", resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, io.LimitReader(resp.Body, postPreviewMax+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > postPreviewMax {
		err = errors.New("prévia grande demais")
	}
	if err != nil {
		_ = os.Remove(dst)
	}
	return err
}

// postCreative creates the ad creative of an existing post. If Meta refuses the
// button on a followers test, it tries once without it (the post still leads to
// the profile).
func (e *executor) postCreative(ctx context.Context, t adtestq.Test, index int, c adtestq.Creative, cta map[string]any) (string, error) {
	var m igMedia
	if err := e.g.get(ctx, c.MediaID, url.Values{"fields": {igMediaFields}}, &m); err != nil {
		return "", fmt.Errorf("post %s: %v", c.Permalink, err)
	}
	if why := m.boostRefusal(); why != "" {
		return "", fmt.Errorf("post %s: %s", c.Permalink, why)
	}
	params := url.Values{
		"name":                      {fmt.Sprintf("%s %d %s", t.ID, index+1, c.Hook)},
		"object_id":                 {e.g.cfg.PageID},
		"instagram_user_id":         {e.g.cfg.IGUserID},
		"source_instagram_media_id": {c.MediaID},
		"call_to_action":            {jsonParam(cta)},
	}
	var created struct {
		ID string `json:"id"`
	}
	err := e.g.post(ctx, e.g.cfg.Account+"/adcreatives", params, &created)
	if err != nil && t.Objective == "seguidores" {
		params.Del("call_to_action")
		params.Del("access_token")
		if retry := e.g.post(ctx, e.g.cfg.Account+"/adcreatives", params, &created); retry == nil {
			err = nil
		}
	}
	if err != nil {
		return "", fmt.Errorf("post %s: %v", c.Permalink, err)
	}
	return created.ID, nil
}

// refusedLine names a creative Meta refused and why, for the owner.
func refusedLine(t adtestq.Test, r adtestq.Refused) string {
	name := fmt.Sprintf("criativo %d", r.Creative+1)
	if r.Creative >= 0 && r.Creative < len(t.Creatives) {
		name += " (" + t.Creatives[r.Creative].Hook + ")"
	}
	return name + ": " + r.Reason
}
