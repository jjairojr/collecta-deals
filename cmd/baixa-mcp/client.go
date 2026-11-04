package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"opdeals/internal/syncqueue"
	"opdeals/internal/trades"
)

var games = []string{"onepiece", "pokemon", "riftbound"}

func validGame(g string) bool {
	for _, k := range games {
		if g == k {
			return true
		}
	}
	return false
}

func (a *app) call(ctx context.Context, method, path string, query url.Values, body, out any) error {
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(data)
	}
	target := a.cfg.OpdealsURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, target, payload)
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

func gameQuery(game string) url.Values {
	return url.Values{"game": {game}}
}

func (a *app) ledger(ctx context.Context, game string) ([]trades.Trade, error) {
	var resp struct {
		Trades []trades.Trade `json:"trades"`
	}
	if err := a.call(ctx, http.MethodGet, "/api/trades", gameQuery(game), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Trades, nil
}

type sale struct {
	Qty          int     `json:"qty"`
	SellPrice    float64 `json:"sellPrice"`
	SellCurrency string  `json:"sellCurrency"`
	SellDate     string  `json:"sellDate"`
	Buyer        string  `json:"buyer"`
	Channel      string  `json:"channel"`
}

func (a *app) sell(ctx context.Context, game, id string, in sale) (trades.Trade, error) {
	var sold trades.Trade
	err := a.call(ctx, http.MethodPost, "/api/trades/"+url.PathEscape(id)+"/sell", gameQuery(game), in, &sold)
	return sold, err
}

func (a *app) createTrade(ctx context.Context, game string, t trades.Trade) (trades.Trade, error) {
	var created trades.Trade
	err := a.call(ctx, http.MethodPost, "/api/trades", gameQuery(game), t, &created)
	return created, err
}

type ligaState struct {
	ID       string  `json:"id"`
	Listed   bool    `json:"ligaListed"`
	Qty      int     `json:"ligaQty"`
	PriceBRL float64 `json:"ligaPriceBRL"`
}

func (a *app) setLiga(ctx context.Context, game string, items []ligaState) (int, error) {
	var resp struct {
		Updated int `json:"updated"`
	}
	err := a.call(ctx, http.MethodPost, "/api/trades/liga", gameQuery(game), map[string][]ligaState{"items": items}, &resp)
	return resp.Updated, err
}

func (a *app) pendingTasks(ctx context.Context) ([]syncqueue.Task, error) {
	var resp struct {
		Tasks []syncqueue.Task `json:"tasks"`
	}
	err := a.call(ctx, http.MethodGet, "/api/syncqueue", url.Values{"status": {"pending"}}, nil, &resp)
	return resp.Tasks, err
}

func (a *app) finishTask(ctx context.Context, id, errMsg string) error {
	path := "/api/syncqueue/" + url.PathEscape(id)
	if errMsg == "" {
		return a.call(ctx, http.MethodPost, path+"/done", nil, struct{}{}, nil)
	}
	return a.call(ctx, http.MethodPost, path+"/error", nil, map[string]string{"error": errMsg}, nil)
}

type ligaOrderItem struct {
	Name string `json:"name"`
	Qty  int    `json:"qty"`
}

type ligaOrder struct {
	Code      string          `json:"code"`
	FirstName string          `json:"firstName"`
	Phone     string          `json:"phone"`
	PaidAt    string          `json:"paidAt"`
	Pickup    bool            `json:"pickup"`
	Shipped   bool            `json:"shipped"`
	Items     []ligaOrderItem `json:"items"`
}

type ligaImport struct {
	Created int `json:"created"`
	Skipped int `json:"skipped"`
}

func (a *app) importLigaOrders(ctx context.Context, orders []ligaOrder) (ligaImport, error) {
	var resp ligaImport
	err := a.call(ctx, http.MethodPost, "/api/postsale/liga-orders", nil, map[string][]ligaOrder{"orders": orders}, &resp)
	return resp, err
}

func (a *app) notify(ctx context.Context, title, body, priority string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.NtfyURL, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Title", title)
	req.Header.Set("Tags", "package")
	req.Header.Set("Priority", priority)
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("não consegui avisar o dono: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("não consegui avisar o dono: ntfy respondeu %d", resp.StatusCode)
	}
	return nil
}
