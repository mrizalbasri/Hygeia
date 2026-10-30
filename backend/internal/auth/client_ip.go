package auth

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ParseTrustedProxyCIDRs accepts explicit proxy networks only. An empty setting
// preserves the default: forwarding headers have no authority.
func ParseTrustedProxyCIDRs(value string) ([]netip.Prefix, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	var prefixes []netip.Prefix
	for _, item := range strings.Split(value, ",") {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(item))
		if err != nil || prefix.Addr().Is4In6() {
			return nil, fmt.Errorf("TRUSTED_PROXY_CIDRS must contain comma-separated IPv4 or IPv6 CIDRs (use native IPv4 CIDRs for mapped addresses)")
		}
		prefixes = append(prefixes, prefix.Masked())
	}
	return prefixes, nil
}

func (h *Handler) trusted(addr netip.Addr) bool {
	for _, prefix := range h.trustedProxies {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

func (h *Handler) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	peer = peer.Unmap().WithZone("")
	fallback := peer.String()
	if !h.trusted(peer) {
		return fallback
	}
	values := r.Header.Values("X-Forwarded-For")
	if len(values) == 0 {
		return fallback
	}
	var chain []netip.Addr
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			addr, err := netip.ParseAddr(strings.TrimSpace(part))
			// XFF contains plain IPs, never ports, zone IDs or hostnames.
			if err != nil || addr.Zone() != "" {
				return fallback
			}
			chain = append(chain, addr.Unmap())
		}
	}
	for i := len(chain) - 1; i >= 0; i-- {
		if !h.trusted(chain[i]) {
			return chain[i].String()
		}
	}
	return fallback
}
