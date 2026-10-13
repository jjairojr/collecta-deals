package collecta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Coupon struct {
	ID         string `json:"id"`
	Code       string `json:"code"`
	ValueCents int64  `json:"valueCents"`
}

type Client interface {
	Reserve(ctx context.Context, userID string, couponIDs []string, totalCents int64) ([]Coupon, error)
	Release(ctx context.Context, codes []string) error
}

type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("collecta: HTTP %d: %s", e.Status, e.Message)
}

func (e *APIError) Rejected() bool {
	switch e.Status {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity:
		return true
	}
	return false
}

type HTTP struct {
	base  string
	token string
	http  *http.Client
}

func New(base, token string) *HTTP {
	return &HTTP{base: strings.TrimRight(base, "/"), token: token, http: &http.Client{Timeout: 8 * time.Second}}
}

func (c *HTTP) Reserve(ctx context.Context, userID string, couponIDs []string, totalCents int64) ([]Coupon, error) {
	var out struct {
		Coupons []Coupon `json:"coupons"`
	}
	if err := c.post(ctx, "/store/coupons/reserve", map[string]any{"userId": userID, "couponIds": couponIDs, "totalCents": totalCents}, &out); err != nil {
		return nil, err
	}
	return out.Coupons, nil
}

func (c *HTTP) Release(ctx context.Context, codes []string) error {
	var out struct {
		OK bool `json:"ok"`
	}
	if err := c.post(ctx, "/store/coupons/release", map[string]any{"codes": codes}, &out); err != nil {
		return err
	}
	if !out.OK {
		return errors.New("collecta: release not acknowledged")
	}
	return nil
}

func (c *HTTP) post(ctx context.Context, path string, body any, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Store-Token", c.token)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("collecta: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("collecta: %w", err)
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return &APIError{Status: resp.StatusCode, Message: messageOf(data)}
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("collecta: HTTP %d: %s", resp.StatusCode, truncate(data, 300))
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("collecta: %w: %s", err, truncate(data, 300))
	}
	return nil
}

func messageOf(data []byte) string {
	var body struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(data, &body); err == nil && body.Message != "" {
		return body.Message
	}
	return "cupom recusado"
}

func truncate(b []byte, n int) string {
	s := strings.TrimSpace(string(b))
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
