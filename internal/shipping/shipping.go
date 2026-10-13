package shipping

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

type Product struct {
	ID             string  `json:"id"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Length         int     `json:"length"`
	Weight         float64 `json:"weight"`
	InsuranceValue float64 `json:"insurance_value"`
	Quantity       int     `json:"quantity"`
}

type Service struct {
	ID           int     `json:"serviceID"`
	Name         string  `json:"name"`
	Company      string  `json:"company"`
	PriceBRL     float64 `json:"priceBRL"`
	DeliveryDays int     `json:"deliveryDays"`
}

type Quoter interface {
	Quote(ctx context.Context, fromCEP, toCEP string, products []Product) ([]Service, error)
}

const (
	PickupServiceID = -1
	JoinServiceID   = -2
)

func Pickup() Service {
	return Service{ID: PickupServiceID, Name: "Retirar em mãos", Company: "Goiânia", PriceBRL: 0}
}

func Join() Service {
	return Service{ID: JoinServiceID, Name: "Juntar com outro pedido", Company: "Collecta", PriceBRL: 0}
}

const (
	ProductionBase = "https://melhorenvio.com.br"
	SandboxBase    = "https://sandbox.melhorenvio.com.br"
	services       = "1,2,17,3"
)

type MelhorEnvio struct {
	base      string
	token     string
	userAgent string
	http      *http.Client
}

func NewMelhorEnvio(base, token, contactEmail string) *MelhorEnvio {
	return &MelhorEnvio{
		base:      base,
		token:     token,
		userAgent: "Collecta (" + contactEmail + ")",
		http:      &http.Client{Timeout: 20 * time.Second},
	}
}

type calculateRequest struct {
	From     postal     `json:"from"`
	To       postal     `json:"to"`
	Products []Product  `json:"products"`
	Options  quoteFlags `json:"options"`
	Services string     `json:"services"`
}

type postal struct {
	PostalCode string `json:"postal_code"`
}

type quoteFlags struct {
	Receipt bool `json:"receipt"`
	OwnHand bool `json:"own_hand"`
}

type calculateResponse struct {
	ID           int         `json:"id"`
	Name         string      `json:"name"`
	Price        json.Number `json:"price"`
	CustomPrice  json.Number `json:"custom_price"`
	DeliveryTime int         `json:"delivery_time"`
	Error        string      `json:"error"`
	Company      struct {
		Name string `json:"name"`
	} `json:"company"`
}

func (m *MelhorEnvio) Quote(ctx context.Context, fromCEP, toCEP string, products []Product) ([]Service, error) {
	body, err := json.Marshal(calculateRequest{
		From:     postal{PostalCode: fromCEP},
		To:       postal{PostalCode: toCEP},
		Products: products,
		Services: services,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.base+"/api/v2/me/shipment/calculate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+m.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", m.userAgent)
	resp, err := m.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("melhor envio: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("melhor envio: HTTP %d: %s", resp.StatusCode, truncate(raw, 300))
	}
	var rows []calculateResponse
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, fmt.Errorf("melhor envio: %w: %s", err, truncate(raw, 300))
	}
	return parseServices(rows), nil
}

func parseServices(rows []calculateResponse) []Service {
	out := make([]Service, 0, len(rows))
	for _, r := range rows {
		if r.Error != "" {
			continue
		}
		price, ok := numberOf(r.CustomPrice)
		if !ok {
			if price, ok = numberOf(r.Price); !ok {
				continue
			}
		}
		if price <= 0 {
			continue
		}
		out = append(out, Service{
			ID:           r.ID,
			Name:         r.Name,
			Company:      r.Company.Name,
			PriceBRL:     price,
			DeliveryDays: r.DeliveryTime,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].PriceBRL < out[j].PriceBRL })
	return out
}

func numberOf(n json.Number) (float64, bool) {
	if n == "" {
		return 0, false
	}
	v, err := n.Float64()
	if err != nil {
		return 0, false
	}
	return v, true
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
