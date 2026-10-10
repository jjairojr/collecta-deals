package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"opdeals/internal/clearance"
	"opdeals/internal/logx"
)

// gateway is a CONNECT proxy that dials upstream over IPv6 only.
//
// Cloudflare denies this network's IPv4 egress for non-browser clients — a
// direct container request gets an "Access denied" page even carrying a valid
// clearance cookie — while the same request over the host's IPv6 address is
// served normally. The container has no IPv6 of its own, so it borrows the
// host's through this gateway.
//
// It must listen on a non-loopback address for the container to reach it, so
// destinations are restricted to the Liga hosts on 443: anyone else who can
// reach the port gets a plain relay to Liga and nothing more.
type gateway struct {
	ln      net.Listener
	log     *logx.Logger
	allowed map[string]bool
	network string
	dialer  *net.Dialer

	// solve, when set, re-earns a host's clearance on request. The scraper calls
	// it after a block instead of the gateway refreshing on a timer: clearance
	// expires in about 30 minutes but a scrape cycle runs every 48 hours, so a
	// timer short enough to matter would drive a browser all day for nothing.
	solve func(host string) (clearance.Entry, error)
}

// newGateway listens on addr and relays to hosts over network, always from
// source. Passing no hosts leaves the relay unrestricted, which is only safe on
// loopback.
func newGateway(addr string, hosts []string, network string, source net.IP, log *logx.Logger) (*gateway, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	var allowed map[string]bool
	if len(hosts) > 0 {
		allowed = make(map[string]bool, len(hosts)*2)
		for _, h := range hosts {
			allowed[strings.ToLower(h)] = true
			allowed[strings.TrimPrefix(strings.ToLower(h), "www.")] = true
		}
	}
	dialer := &net.Dialer{Timeout: 20 * time.Second}
	if source != nil {
		dialer.LocalAddr = &net.TCPAddr{IP: source}
	}
	return &gateway{ln: ln, log: log, allowed: allowed, network: network, dialer: dialer}, nil
}

func (g *gateway) addr() string { return g.ln.Addr().String() }

func (g *gateway) close() error { return g.ln.Close() }

func (g *gateway) serve() {
	for {
		conn, err := g.ln.Accept()
		if err != nil {
			return
		}
		go g.handle(conn)
	}
}

// serveControl answers the one non-CONNECT route the gateway exposes:
// POST /solve?host=… re-earns that host's clearance and returns once the file
// on disk holds a fresh cookie, so the caller can retry immediately.
func (g *gateway) serveControl(client net.Conn, req *http.Request) {
	if req.URL.Path != "/solve" || g.solve == nil {
		fmt.Fprint(client, "HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\n\r\n")
		return
	}
	host := strings.ToLower(req.URL.Query().Get("host"))
	if host == "" || (g.allowed != nil && !g.allowed[host]) {
		g.log.Printf("solve refused for %q", host)
		fmt.Fprint(client, "HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\n\r\n")
		return
	}
	entry, err := g.solve(host)
	if err != nil {
		g.log.Printf("solve %s: %v", host, err)
		fmt.Fprint(client, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\n\r\n")
		return
	}
	body, err := json.Marshal(entry)
	if err != nil {
		body = []byte("{}")
	}
	fmt.Fprintf(client, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
}

func (g *gateway) handle(client net.Conn) {
	defer client.Close()

	br := bufio.NewReader(client)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	if req.Method != http.MethodConnect {
		g.serveControl(client, req)
		return
	}

	target := req.URL.Host
	if !strings.Contains(target, ":") {
		target += ":443"
	}
	host, port, err := net.SplitHostPort(target)
	if err != nil || (g.allowed != nil && (port != "443" || !g.allowed[strings.ToLower(host)])) {
		g.log.Printf("refused %s", target)
		fmt.Fprint(client, "HTTP/1.1 403 Forbidden\r\n\r\n")
		return
	}

	upstream, err := g.dialer.Dial(g.network, target)
	if err != nil && g.allowed == nil {
		// The challenge pulls in Cloudflare and Google hosts that have no AAAA
		// record; only the Liga hosts have to hold the pinned identity.
		upstream, err = net.DialTimeout("tcp", target, 20*time.Second)
	}
	if err != nil {
		g.log.Printf("dial %s %s: %v", g.network, target, err)
		fmt.Fprint(client, "HTTP/1.1 502 Bad Gateway\r\n\r\n")
		return
	}
	defer upstream.Close()

	if _, err := fmt.Fprint(client, "HTTP/1.1 200 Connection Established\r\n\r\n"); err != nil {
		return
	}
	if buffered := br.Buffered(); buffered > 0 {
		if _, err := io.CopyN(upstream, br, int64(buffered)); err != nil {
			return
		}
	}

	done := make(chan struct{}, 2)
	go func() { io.Copy(upstream, client); done <- struct{}{} }()
	go func() { io.Copy(client, upstream); done <- struct{}{} }()
	<-done
}
