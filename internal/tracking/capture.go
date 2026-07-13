package tracking

import (
	"context"
	"fmt"
	"time"

	"opdeals/internal/liga"
	"opdeals/internal/logx"
	"opdeals/internal/model"
)

const (
	dateLayout  = "2006-01-02"
	stampLayout = "2006-01-02T15"
	sealedSet   = "SEALED"
)

type Capturer struct {
	log        *logx.Logger
	liga       *liga.Client
	store      *Store
	sets       []string
	minCardBRL float64
	tz         *time.Location
	interval   time.Duration
	fxProvider func() float64
	sealed     bool
	floorLangs []string
}

func NewCapturer(log *logx.Logger, ligaClient *liga.Client, store *Store, sets []string, minCardBRL float64, tz *time.Location, interval time.Duration, sealed bool, floorLangs []string, fxProvider func() float64) *Capturer {
	return &Capturer{log: log, liga: ligaClient, store: store, sets: sets, minCardBRL: minCardBRL, tz: tz, interval: interval, sealed: sealed, floorLangs: floorLangs, fxProvider: fxProvider}
}

func (c *Capturer) Sets() []string { return c.sets }

func (c *Capturer) Set() string {
	if len(c.sets) > 0 {
		return c.sets[0]
	}
	return ""
}

// DateFor returns the storage key for the snapshot covering now. With an
// interval of a day or more it is the calendar date (backward-compatible with
// the original daily model); with a sub-day interval it is the date plus the
// hour of the aligned time-slot, so multiple snapshots per day get distinct keys.
func (c *Capturer) DateFor(now time.Time) string {
	t := now.In(c.tz)
	if c.interval <= 0 || c.interval >= 24*time.Hour {
		return t.Format(dateLayout)
	}
	midnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	slots := int(t.Sub(midnight) / c.interval)
	return midnight.Add(time.Duration(slots) * c.interval).Format(stampLayout)
}

func (c *Capturer) CaptureSealed(ctx context.Context, now time.Time) error {
	date := c.DateFor(now)
	fx := 0.0
	if c.fxProvider != nil {
		fx = c.fxProvider()
	}
	c.log.Printf("tracking: starting SEALED capture for %s", date)
	if err := c.captureSealed(ctx, now, date, fx); err != nil {
		c.log.Printf("tracking: sealed capture failed: %v", err)
		return err
	}
	return nil
}

func (c *Capturer) CaptureSingles(ctx context.Context, now time.Time) error {
	date := c.DateFor(now)
	fx := 0.0
	if c.fxProvider != nil {
		fx = c.fxProvider()
	}
	c.log.Printf("tracking: starting SINGLES capture for %s (%d sets)", date, len(c.sets))
	if err := c.captureSingles(ctx, now, date, fx); err != nil {
		c.log.Printf("tracking: singles capture failed: %v", err)
		return err
	}
	return nil
}

func (c *Capturer) Capture(ctx context.Context, now time.Time) error {
	if err := c.capture(ctx, now); err != nil {
		c.log.Printf("tracking: capture failed: %v", err)
		return err
	}
	return nil
}

func (c *Capturer) capture(ctx context.Context, now time.Time) error {
	date := c.DateFor(now)
	fx := 0.0
	if c.fxProvider != nil {
		fx = c.fxProvider()
	}
	singlesErr := c.captureSingles(ctx, now, date, fx)
	if singlesErr != nil {
		c.log.Printf("tracking: singles capture: %v", singlesErr)
	}
	if c.sealed && !c.store.HasDay(sealedSet, date) {
		if err := c.captureSealed(ctx, now, date, fx); err != nil {
			c.log.Printf("tracking: sealed capture: %v", err)
		}
	}
	return singlesErr
}

func (c *Capturer) captureSingles(ctx context.Context, now time.Time, date string, fx float64) error {
	wanted := map[string]bool{}
	for _, set := range c.sets {
		if !c.store.HasDay(set, date) {
			wanted[set] = true
		}
	}
	if len(wanted) == 0 {
		c.log.Printf("tracking: all sets already captured for %s", date)
		return nil
	}

	listings, err := c.liga.Listings(ctx)
	if err != nil {
		return fmt.Errorf("tracking listings: %w", err)
	}
	bySet := map[string][]model.BrazilListing{}
	toDetail := make([]model.BrazilListing, 0, len(listings))
	for _, l := range listings {
		if !wanted[l.SetCode] || l.LowBRL < c.minCardBRL {
			continue
		}
		bySet[l.SetCode] = append(bySet[l.SetCode], l)
		toDetail = append(toDetail, l)
	}
	if len(toDetail) == 0 {
		return fmt.Errorf("tracking: no listings fetched for sets %v", keysOf(wanted))
	}

	detail := c.liga.StockDetail(ctx, toDetail)

	for set := range wanted {
		setListings := bySet[set]
		if len(setListings) == 0 {
			c.log.Printf("tracking: no %s listings; skipping", set)
			continue
		}
		cards := make([]CardDay, 0, len(setListings))
		for _, l := range setListings {
			stores := detail[l.URL]
			low := l.LowBRL
			if f, ok := liga.Floor(stores, c.floorLangs); ok && f > 0 {
				low = f
			}
			cards = append(cards, CardDay{
				Number: l.Number,
				Name:   l.Name,
				LowBRL: low,
				AvgBRL: l.AvgBRL,
				URL:    l.URL,
				Stores: toStoreQty(stores),
			})
		}
		day := DaySnapshot{
			Set:        set,
			Date:       date,
			CapturedAt: now.In(c.tz),
			FXRate:     fx,
			Cards:      cards,
		}
		if err := c.store.SaveDay(day); err != nil {
			return fmt.Errorf("tracking save %s: %w", set, err)
		}
		c.log.Printf("tracking: captured %s %s (%d cards)", set, date, len(cards))
	}
	return nil
}

func (c *Capturer) captureSealed(ctx context.Context, now time.Time, date string, fx float64) error {
	products, err := c.liga.SealedListings(ctx)
	if err != nil {
		return fmt.Errorf("sealed listings: %w", err)
	}
	if len(products) == 0 {
		return fmt.Errorf("sealed: no products enumerated")
	}
	detail := c.liga.SealedDetail(ctx, products)

	cards := make([]CardDay, 0, len(products))
	for _, p := range products {
		stores := detail[p.URL]
		if len(stores) == 0 {
			continue
		}
		low := minStorePrice(stores)
		if f, ok := liga.Floor(stores, c.floorLangs); ok && f > 0 {
			low = f
		}
		cards = append(cards, CardDay{
			Number: p.PCode,
			Name:   p.Name,
			LowBRL: low,
			URL:    p.URL,
			Stores: toStoreQty(stores),
		})
	}
	if len(cards) == 0 {
		return fmt.Errorf("sealed: no per-store stock decoded")
	}
	day := DaySnapshot{
		Set:        sealedSet,
		Date:       date,
		CapturedAt: now.In(c.tz),
		FXRate:     fx,
		Cards:      cards,
	}
	if err := c.store.SaveDay(day); err != nil {
		return fmt.Errorf("sealed save: %w", err)
	}
	c.log.Printf("tracking: captured %s %s (%d products)", sealedSet, date, len(cards))
	return nil
}

func minStorePrice(listings []liga.StoreListing) float64 {
	low := 0.0
	for _, l := range listings {
		if !l.PriceKnown || l.PriceBRL <= 0 {
			continue
		}
		if low == 0 || l.PriceBRL < low {
			low = l.PriceBRL
		}
	}
	return low
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func toStoreQty(listings []liga.StoreListing) []StoreQty {
	out := make([]StoreQty, 0, len(listings))
	for _, l := range listings {
		out = append(out, StoreQty{
			StoreID:    l.StoreID,
			StoreName:  l.StoreName,
			Condition:  l.Condition,
			Language:   l.Language,
			Quantity:   l.Quantity,
			Known:      l.QtyKnown,
			PriceBRL:   l.PriceBRL,
			PriceKnown: l.PriceKnown,
		})
	}
	return out
}
