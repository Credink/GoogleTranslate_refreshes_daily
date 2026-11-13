// Package netcheck probes local network capabilities (e.g. IPv6 usability).
package netcheck

import (
	"context"
	"net"
	"time"
)

// IPv6Usable reports whether this machine appears able to use IPv6 for outbound traffic.
// Checks for a non-loopback global-ish IPv6 address, then tries a short TCP dial to a
// well-known IPv6 endpoint (Google DNS). Failures mean "treat this run as IPv4-only".
func IPv6Usable(ctx context.Context) bool {
	if !hasNonLoopbackIPv6() {
		return false
	}
	d := net.Dialer{Timeout: 1500 * time.Millisecond}
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	conn, err := d.DialContext(cctx, "tcp6", "[2001:4860:4860::8888]:443")
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func hasNonLoopbackIPv6() bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.To4() != nil {
				continue
			}
			if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				continue
			}
			return true
		}
	}
	return false
}
