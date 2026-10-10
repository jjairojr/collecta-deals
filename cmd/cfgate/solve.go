package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"

	"opdeals/internal/clearance"
	"opdeals/internal/game"
	"opdeals/internal/logx"
)

const (
	pollInterval  = 2 * time.Second
	settleAfter   = 1500 * time.Millisecond
	perHostBudget = 90 * time.Second
)

// solveAll drives one headed Chrome through every host's challenge and returns
// the clearance each one issued. Cloudflare rejects every headless browser we
// can run (FlareSolverr, Byparr and headless chromedp all fail against these
// hosts), so this is the one step that cannot live in the container.
func solveAll(ctx context.Context, hosts []string, budget time.Duration, solveProxy string, headless, visible bool, log *logx.Logger) (map[string]clearance.Entry, []string) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("lang", "pt-BR"),
		chromedp.WindowSize(1280, 900),
	)
	if solveProxy != "" {
		opts = append(opts, chromedp.Flag("proxy-server", solveProxy))
	}
	if !visible && !headless {
		opts = append(opts, chromedp.Flag("window-position", "-3000,-3000"))
	}
	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	entries := map[string]clearance.Entry{}
	var failed []string
	for _, host := range hosts {
		entry, err := solve(ctx, browserCtx, host, budget)
		if err != nil {
			log.Printf("%s: %v", host, err)
			failed = append(failed, host)
			continue
		}
		entries[host] = entry
		log.Printf("%s solved (ua %s)", host, shortUA(entry.UserAgent))
	}
	return entries, failed
}

func solve(ctx context.Context, browserCtx context.Context, host string, budget time.Duration) (clearance.Entry, error) {
	tabCtx, tabCancel := chromedp.NewContext(browserCtx)
	defer tabCancel()

	target := "https://" + host + "/"
	if err := chromedp.Run(tabCtx, chromedp.Navigate(target)); err != nil {
		return clearance.Entry{}, fmt.Errorf("navigate: %w", err)
	}

	deadline := time.Now().Add(budget)
	var title string
	for time.Now().Before(deadline) {
		if err := chromedp.Run(tabCtx, chromedp.Title(&title)); err == nil && title != "" && !isChallenge(title) {
			break
		}
		select {
		case <-ctx.Done():
			return clearance.Entry{}, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
	if title == "" || isChallenge(title) {
		return clearance.Entry{}, fmt.Errorf("challenge not cleared (last title %q)", title)
	}

	var ua, cookie string
	err := chromedp.Run(tabCtx,
		chromedp.Sleep(settleAfter),
		chromedp.Evaluate("navigator.userAgent", &ua),
		chromedp.ActionFunc(func(ctx context.Context) error {
			cookies, err := network.GetCookies().Do(ctx)
			if err != nil {
				return err
			}
			for _, c := range cookies {
				if c.Name == "cf_clearance" && strings.HasSuffix(host, strings.TrimPrefix(c.Domain, ".")) {
					cookie = c.Value
				}
			}
			return nil
		}),
	)
	if err != nil {
		return clearance.Entry{}, err
	}
	if cookie == "" {
		return clearance.Entry{}, fmt.Errorf("page loaded (%q) but no cf_clearance cookie was set", title)
	}
	if ua == "" {
		return clearance.Entry{}, fmt.Errorf("could not read the browser user agent")
	}
	return clearance.Entry{Cookie: cookie, UserAgent: ua, ObtainedAt: time.Now()}, nil
}

// merge keeps hosts that were not solved in this run, so a partial refresh never
// drops a cookie that is still good.
func merge(store *clearance.Store, fresh map[string]clearance.Entry, hosts []string) error {
	combined := map[string]clearance.Entry{}
	for _, host := range hosts {
		if e, ok := store.Lookup(host); ok {
			combined[host] = e
		}
	}
	for host, e := range fresh {
		combined[host] = e
	}
	return store.Save(combined)
}

func targetHosts(flagValue string) []string {
	if strings.TrimSpace(flagValue) == "" {
		return allHosts()
	}
	var hosts []string
	for _, h := range strings.Split(flagValue, ",") {
		if h = strings.TrimSpace(h); h != "" {
			hosts = append(hosts, h)
		}
	}
	return hosts
}

// allHosts is one host per challenged game — the cookie covers www and apex
// alike, so only the canonical host needs solving.
func allHosts() []string {
	seen := map[string]bool{}
	var hosts []string
	for _, g := range game.All() {
		if !g.Challenged || len(g.LigaHosts) == 0 {
			continue
		}
		h := g.LigaHosts[0]
		if seen[h] {
			continue
		}
		seen[h] = true
		hosts = append(hosts, h)
	}
	return hosts
}

func isChallenge(title string) bool {
	t := strings.ToLower(title)
	return strings.Contains(t, "just a moment") || strings.Contains(t, "moment...")
}

func shortUA(ua string) string {
	if i := strings.Index(ua, "Chrome/"); i >= 0 {
		rest := ua[i:]
		if j := strings.Index(rest, " "); j > 0 {
			return rest[:j]
		}
		return rest
	}
	return ua
}
