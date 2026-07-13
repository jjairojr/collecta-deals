package fx

import (
	"context"
	"encoding/json"
	"fmt"

	"opdeals/internal/httpx"
	"opdeals/internal/logx"
)

const endpoint = "https://api.frankfurter.app/latest?from=BRL&to=USD"

type response struct {
	Rates struct {
		USD float64 `json:"USD"`
	} `json:"rates"`
}

func Rate(ctx context.Context, client *httpx.Client, logger *logx.Logger, override float64) (float64, error) {
	if override > 0 {
		logger.Printf("FX  BRL->USD = %.4f (override)", override)
		return override, nil
	}
	body, err := client.Get(ctx, endpoint)
	if err != nil {
		return 0, fmt.Errorf("fx fetch: %w", err)
	}
	var r response
	if err := json.Unmarshal(body, &r); err != nil {
		return 0, fmt.Errorf("fx decode: %w", err)
	}
	if r.Rates.USD <= 0 {
		return 0, fmt.Errorf("fx rate unavailable")
	}
	logger.Printf("FX  BRL->USD = %.4f (frankfurter.app)", r.Rates.USD)
	return r.Rates.USD, nil
}
