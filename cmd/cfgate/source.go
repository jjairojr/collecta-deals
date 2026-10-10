package main

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// pickStableIPv6 returns a global IPv6 address that macOS will not rotate.
//
// Privacy extensions give the interface a rolling set of `temporary` addresses,
// and outbound connections pick whichever is current. Cloudflare binds
// cf_clearance to the address that solved the challenge, so a cookie earned on
// one temporary address stops working the moment macOS rotates to the next —
// which is why a freshly solved cookie can die within minutes. The `secured`
// autoconf address lives as long as the ISP prefix does, so pinning it keeps
// the browser and the scraper on one identity.
//
// The `temporary`/`secured`/`deprecated` flags are not exposed by net.Interface,
// so this reads ifconfig. Returns nil when no such address exists (IPv4-only
// network), which leaves the dialer on the system default.
func pickStableIPv6(iface string) (net.IP, error) {
	out, err := exec.Command("ifconfig", iface).Output()
	if err != nil {
		return nil, fmt.Errorf("ifconfig %s: %w", iface, err)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 || fields[0] != "inet6" {
			continue
		}
		flags := strings.Join(fields[2:], " ")
		if strings.Contains(flags, "temporary") || strings.Contains(flags, "deprecated") {
			continue
		}
		ip := net.ParseIP(strings.Split(fields[1], "%")[0])
		if ip == nil || ip.To4() != nil || !ip.IsGlobalUnicast() {
			continue
		}
		return ip, nil
	}
	return nil, nil
}

// defaultInterface reports the interface carrying the default IPv6 route.
func defaultInterface() string {
	out, err := exec.Command("route", "-n", "get", "-inet6", "default").Output()
	if err != nil {
		return "en0"
	}
	for _, line := range strings.Split(string(out), "\n") {
		if _, rest, ok := strings.Cut(line, "interface:"); ok {
			return strings.TrimSpace(rest)
		}
	}
	return "en0"
}
