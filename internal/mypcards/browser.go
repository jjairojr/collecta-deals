package mypcards

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

const realUA = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"

const (
	challengeTimeout = 50 * time.Second
	fetchTimeout     = 45 * time.Second
)

// browser owns a single headless Chrome whose cf_clearance cookie (obtained once
// by solve) is shared by every tab opened from b.ctx. It is tied to the context
// passed to newBrowser, so cancelling that context tears the browser down.
type browser struct {
	allocCancel context.CancelFunc
	ctxCancel   context.CancelFunc
	ctx         context.Context
}

func newBrowser(ctx context.Context) *browser {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("lang", "pt-BR"),
		chromedp.UserAgent(realUA),
		chromedp.WindowSize(1280, 900),
	)
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	browserCtx, ctxCancel := chromedp.NewContext(allocCtx)
	return &browser{allocCancel: allocCancel, ctxCancel: ctxCancel, ctx: browserCtx}
}

func (b *browser) close() {
	b.ctxCancel()
	b.allocCancel()
}

// solve loads the home page and waits for the Cloudflare JS challenge to clear,
// establishing the cf_clearance cookie that all subsequent tabs reuse. It runs
// directly on b.ctx so the browser, started by this first navigation, is owned
// by the browser context rather than a short-lived derived one.
func (b *browser) solve(ctx context.Context) error {
	if err := chromedp.Run(b.ctx, chromedp.Navigate(baseURL+"/")); err != nil {
		return fmt.Errorf("navigate home: %w", err)
	}
	deadline := time.Now().Add(challengeTimeout)
	var title string
	for time.Now().Before(deadline) {
		if err := chromedp.Run(b.ctx, chromedp.Title(&title)); err == nil && title != "" && !isChallenge(title) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-b.ctx.Done():
			return b.ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	return fmt.Errorf("cloudflare challenge not cleared (last title %q)", title)
}

// fetch navigates a fresh tab (sharing the solved session) to url and returns
// its HTML. A watchdog cancels the tab if fetchTimeout elapses.
func (b *browser) fetch(_ context.Context, url string) (string, error) {
	tabCtx, tabCancel := chromedp.NewContext(b.ctx)
	defer tabCancel()
	timer := time.AfterFunc(fetchTimeout, tabCancel)
	defer timer.Stop()

	var doc, title string
	err := chromedp.Run(tabCtx,
		chromedp.Navigate(url),
		chromedp.WaitReady("body"),
		chromedp.Sleep(700*time.Millisecond),
		chromedp.Title(&title),
		chromedp.OuterHTML("html", &doc),
	)
	if err != nil {
		return "", err
	}
	if isChallenge(title) {
		// Re-challenged: wait briefly and re-read once.
		_ = chromedp.Run(tabCtx, chromedp.Sleep(4*time.Second), chromedp.OuterHTML("html", &doc))
	}
	return doc, nil
}

func isChallenge(title string) bool {
	return strings.Contains(strings.ToLower(title), "just a moment") ||
		strings.Contains(strings.ToLower(title), "moment...")
}
