package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const (
	tunnelStartMax  = 60 * time.Second
	tunnelReadyMax  = 60 * time.Second
	tunnelReadyPoll = 2 * time.Second
)

var tunnelURL = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

const tunnelUp = "Registered tunnel connection"

type videoHost interface {
	share(ctx context.Context, path string) (string, func(), error)
}

type tunnelHost struct {
	binary string
}

func publicDNSClient() *http.Client {
	resolver := &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
		var d net.Dialer
		return d.DialContext(ctx, network, "1.1.1.1:53")
	}}
	dialer := &net.Dialer{Resolver: resolver, Timeout: 10 * time.Second}
	return &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{DialContext: dialer.DialContext}}
}

func (h tunnelHost) share(ctx context.Context, path string) (string, func(), error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	name := "/" + randomID() + ".mp4"
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != name {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "video/mp4")
		http.ServeFile(w, r, path)
	})}
	go func() { _ = srv.Serve(ln) }()
	tctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(tctx, h.binary, "tunnel", "--url", "http://"+ln.Addr().String(), "--no-autoupdate")
	logs, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		_ = srv.Close()
		return "", nil, err
	}
	stop := func() {
		cancel()
		_ = cmd.Wait()
		_ = srv.Close()
	}
	if err := cmd.Start(); err != nil {
		cancel()
		_ = srv.Close()
		return "", nil, fmt.Errorf("cloudflared não abriu: %v", err)
	}
	base, err := waitTunnelURL(ctx, logs)
	if err != nil {
		stop()
		return "", nil, err
	}
	link := base + name
	if err := h.waitReachable(ctx, link); err != nil {
		stop()
		return "", nil, err
	}
	return link, stop, nil
}

func waitTunnelURL(ctx context.Context, logs io.Reader) (string, error) {
	found := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(logs)
		link, sent := "", false
		for sc.Scan() {
			line := sc.Text()
			if m := tunnelURL.FindString(line); m != "" && link == "" {
				link = m
			}
			if link != "" && !sent && strings.Contains(line, tunnelUp) {
				found <- link
				sent = true
			}
		}
	}()
	select {
	case u := <-found:
		return u, nil
	case <-time.After(tunnelStartMax):
		return "", errors.New("o túnel do cloudflared não devolveu o link a tempo")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (h tunnelHost) waitReachable(ctx context.Context, link string) error {
	client := publicDNSClient()
	deadline := time.Now().Add(tunnelReadyMax)
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, link, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		if time.Now().After(deadline) {
			return errors.New("o link do túnel não respondeu a tempo")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(tunnelReadyPoll):
		}
	}
}
