// Command cfgate is the host half of the Liga scraper's Cloudflare bypass.
//
// It does two things the container cannot: it drives a headed Chrome to earn
// the cf_clearance cookies (every headless browser is rejected), and it serves
// an IPv6 CONNECT gateway for the scraper's requests (Cloudflare denies this
// network's IPv4 egress for non-browser clients, and the container has no IPv6).
//
// Clearance is earned on demand — the scraper asks for it after a block — so an
// idle stack costs nothing. Run it alongside the stack; the container is pointed
// at it with -cf-clearance and -liga-proxy.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"opdeals/internal/clearance"
	"opdeals/internal/logx"
)

var errNoneSolved = errors.New("no host solved")

func unsolvedError(hosts []string) error {
	return fmt.Errorf("unsolved: %s", strings.Join(hosts, ", "))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	out := flag.String("out", "data/cf-clearance.json", "path of the clearance file to write")
	hostsFlag := flag.String("hosts", "", "comma-separated hosts to solve (default: every challenged game's Liga host)")
	budget := flag.Duration("solve-timeout", perHostBudget, "max wait per host for the challenge to clear")
	refresh := flag.Duration("refresh-interval", 0, "re-solve every host on this interval as a safety net (0 = only on demand, which is what the scraper asks for)")
	listen := flag.String("listen", "0.0.0.0:8891", "address the IPv6 CONNECT gateway listens on (must be reachable from the container, so not loopback)")
	network := flag.String("dial", "tcp6", "network the gateway dials upstream with")
	once := flag.Bool("once", false, "solve every host now, write the file and exit")
	sourceFlag := flag.String("source", "", "IPv6 address every request egresses from (empty = the interface's stable, non-temporary address)")
	headless := flag.Bool("headless", false, "run Chrome headless (Cloudflare currently rejects this; for testing only)")
	visible := flag.Bool("visible", false, "keep the Chrome window on-screen instead of parking it off-screen")
	flag.Parse()

	log := logx.New(os.Stderr).WithPrefix("CFGATE")
	hosts := targetHosts(*hostsFlag)
	if len(hosts) == 0 {
		return fmt.Errorf("no hosts to solve")
	}
	store := clearance.NewStore(*out)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	source, err := resolveSource(*sourceFlag)
	if err != nil {
		return err
	}
	if source == nil {
		log.Printf("no stable IPv6 address found; egress will follow the system default and clearance will expire whenever it rotates")
	} else {
		log.Printf("pinning egress to %s", source)
	}

	// The browser goes through the same pinned source as the scraper, so the
	// address that earns the clearance is the address that later presents it.
	// Its relay is unrestricted (the challenge pulls in Cloudflare's own hosts)
	// and therefore loopback-only.
	solveGW, err := newGateway("127.0.0.1:0", nil, *network, source, log)
	if err != nil {
		return fmt.Errorf("solve gateway: %w", err)
	}
	defer solveGW.close()
	go solveGW.serve()
	solveProxy := "http://" + solveGW.addr()

	ref := newRefresher(store, hosts, *budget, *headless, *visible, log)

	if *once {
		return ref.all(ctx, solveProxy)
	}

	gw, err := newGateway(*listen, hosts, *network, source, log)
	if err != nil {
		return fmt.Errorf("gateway: %w", err)
	}
	gw.solve = func(host string) (clearance.Entry, error) {
		if err := ref.one(ctx, solveProxy, host); err != nil {
			return clearance.Entry{}, err
		}
		e, _ := store.Lookup(host)
		return e, nil
	}
	defer gw.close()
	go gw.serve()
	log.Printf("gateway on %s dialing %s, restricted to %d Liga hosts; clearance solved on demand", gw.addr(), *network, len(hosts))

	if *refresh <= 0 {
		<-ctx.Done()
		log.Printf("shutting down")
		return nil
	}

	ticker := time.NewTicker(*refresh)
	defer ticker.Stop()
	log.Printf("also re-solving every %s as a safety net", *refresh)
	for {
		select {
		case <-ctx.Done():
			log.Printf("shutting down")
			return nil
		case <-ticker.C:
			if err := ref.all(ctx, solveProxy); err != nil {
				log.Printf("refresh: %v", err)
			}
		}
	}
}

// resolveSource returns the egress address to pin, from the flag when given and
// otherwise from the interface carrying the default IPv6 route.
func resolveSource(flagValue string) (net.IP, error) {
	if strings.TrimSpace(flagValue) != "" {
		ip := net.ParseIP(strings.TrimSpace(flagValue))
		if ip == nil {
			return nil, fmt.Errorf("parse source %q", flagValue)
		}
		return ip, nil
	}
	return pickStableIPv6(defaultInterface())
}
